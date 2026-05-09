// bintool-vendor reads a vendored-tool descriptor (vendor/<name>.yaml),
// downloads the upstream binary for darwin/linux × amd64/arm64,
// verifies SHA256 checksums, extracts each binary, and shells out to
// `otters bin build` (and optionally `otters bin push`) to package the
// result as an OCI BIN image.
//
// The descriptor schema supports multiple versions per tool. Top-level
// fields are defaults; each entry under `versions:` may override any
// of them. Each version is published as its own image with one tag per
// alias declared on it. The first entry in `versions:` implicitly gets
// the `latest` alias if no other version claims it.
//
// Usage:
//
//	bintool-vendor -descriptor vendor/jq.yaml -registry ghcr.io/openotters/tools \
//	    [-version 1.8.1] [-out /tmp/otters-bin] [-otters otters] [-push]
package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"
)

// Descriptor is one tool's full vendor manifest. Top-level fields are
// shared defaults; per-version overrides live in `Versions[]`. Maps
// (`OSAlias`, `ArchAlias`, `Checksums`) replace wholesale, not merge.
type Descriptor struct {
	// Identity (top-level only — versions can't change the tool name).
	Name string `yaml:"name"`

	// Defaults inherited by every version unless overridden.
	Description     string            `yaml:"description"`
	Source          string            `yaml:"source"`
	Usage           string            `yaml:"usage"`
	URLTemplate     string            `yaml:"url_template"`
	Archive         string            `yaml:"archive"` // "tar.gz" | "tar" | "zip" | "raw"
	BinaryInArchive string            `yaml:"binary_in_archive"`
	OSAlias         map[string]string `yaml:"os_alias"`
	ArchAlias       map[string]string `yaml:"arch_alias"`

	Versions []VersionEntry `yaml:"versions"`
}

// VersionEntry overrides the descriptor defaults for one version.
// Every field is optional — empty values inherit the top-level
// descriptor — except Version and Checksums.
type VersionEntry struct {
	Version string   `yaml:"version"`
	Aliases []string `yaml:"aliases"`

	Description     string            `yaml:"description"`
	Usage           string            `yaml:"usage"`
	URLTemplate     string            `yaml:"url_template"`
	Archive         string            `yaml:"archive"`
	BinaryInArchive string            `yaml:"binary_in_archive"`
	OSAlias         map[string]string `yaml:"os_alias"`
	ArchAlias       map[string]string `yaml:"arch_alias"`

	Checksums map[string]string `yaml:"checksums"` // key: "<os>/<arch>" (Go convention)

	// Per-platform overrides for upstreams that publish different
	// archive types or URL paths per platform (gh ships .tar.gz on
	// linux but .zip on macOS; pandoc has arch-specific paths). Keys
	// are "<os>/<arch>" using Go conventions. Each value can override
	// url_template / archive / binary_in_archive and supplies an
	// optional `target` string injected as {{.Target}} into the
	// templates (used by Rust tools whose URLs/paths embed the target
	// triple, e.g. aarch64-apple-darwin).
	Platforms map[string]PlatformEntry `yaml:"platforms"`
}

// PlatformEntry overrides the version defaults for one (os, arch).
// Every field is optional — empty values inherit from the version.
type PlatformEntry struct {
	Target          string `yaml:"target"` // injected as {{.Target}} in templates
	URLTemplate     string `yaml:"url_template"`
	Archive         string `yaml:"archive"`
	BinaryInArchive string `yaml:"binary_in_archive"`
}

// Resolved is one version flattened into the same shape the
// download/extract/build code consumes — defaults already merged in,
// final tag list computed. URLTemplate / Archive / BinaryInArchive
// here are the version-level defaults; per-platform overrides are
// applied at fetch time via Platform().
type Resolved struct {
	Name            string
	Description     string
	Source          string
	Usage           string
	Version         string
	URLTemplate     string
	Archive         string
	BinaryInArchive string
	OSAlias         map[string]string
	ArchAlias       map[string]string
	Checksums       map[string]string
	Platforms       map[string]PlatformEntry
	Tags            []string
}

// PlatformResolved is the effective configuration for one (os, arch)
// pair after platform overrides land on top of version defaults.
type PlatformResolved struct {
	URLTemplate     string
	Archive         string
	BinaryInArchive string
	Target          string
	Checksum        string
}

// Platform resolves the version's defaults plus any override under
// `versions[].platforms[<os>/<arch>]` into a single concrete config.
func (r *Resolved) Platform(osName, arch string) (PlatformResolved, error) {
	key := osName + "/" + arch
	checksum, ok := r.Checksums[key]
	if !ok {
		return PlatformResolved{}, fmt.Errorf("missing checksum for %s", key)
	}
	pr := PlatformResolved{
		URLTemplate:     r.URLTemplate,
		Archive:         r.Archive,
		BinaryInArchive: r.BinaryInArchive,
		Checksum:        checksum,
	}
	if p, ok := r.Platforms[key]; ok {
		if p.URLTemplate != "" {
			pr.URLTemplate = p.URLTemplate
		}
		if p.Archive != "" {
			pr.Archive = p.Archive
		}
		if p.BinaryInArchive != "" {
			pr.BinaryInArchive = p.BinaryInArchive
		}
		pr.Target = p.Target
	}
	return pr, nil
}

// Platforms we always build for. Matches the existing tools:publish target.
var platforms = []struct {
	OS, Arch string
}{
	{"darwin", "arm64"},
	{"darwin", "amd64"},
	{"linux", "arm64"},
	{"linux", "amd64"},
}

func main() {
	var (
		descriptorPath = flag.String("descriptor", "", "Path to the vendored-tool descriptor YAML")
		versionRef     = flag.String("version", "", "Publish only this version (default: every version in the descriptor)")
		registry       = flag.String("registry", "ghcr.io/openotters/tools", "Image registry prefix")
		outDir         = flag.String("out", "/tmp/otters-bin", "Working directory for downloaded binaries")
		ottersBin      = flag.String("otters", "otters", "Path to the otters CLI")
		push           = flag.Bool("push", false, "Also push the image after building")
	)
	flag.Parse()

	if *descriptorPath == "" {
		fatal("missing -descriptor")
	}

	d, err := loadDescriptor(*descriptorPath)
	if err != nil {
		fatal("loading descriptor: %v", err)
	}

	if err := d.validate(); err != nil {
		fatal("invalid descriptor: %v", err)
	}

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		fatal("creating out dir: %v", err)
	}

	resolved, err := d.resolveVersions(*versionRef)
	if err != nil {
		fatal("resolving versions: %v", err)
	}

	for _, r := range resolved {
		if err := publish(r, *registry, *outDir, *ottersBin, *push); err != nil {
			fatal("publishing %s@%s: %v", r.Name, r.Version, err)
		}
	}

	fmt.Printf("ok — %s: %d version(s) built%s\n", d.Name, len(resolved), pushedNote(*push))
}

// publish handles one resolved version: download per platform, extract,
// shell out to `otters bin build` with all platform binaries + every
// alias as a separate -t flag, optionally push each tag.
func publish(r Resolved, registry, outDir, ottersBin string, push bool) error {
	binaries := make(map[string]string, len(platforms))

	// Iterate the canonical platform list but skip any platform the
	// descriptor doesn't have a checksum for. Some upstreams have
	// dropped less-popular platforms (fd ships no x86_64 macOS as of
	// v10) — we don't want one missing target to block shipping the
	// rest. At least one platform must be present.
	for _, p := range platforms {
		key := p.OS + "/" + p.Arch
		if _, ok := r.Checksums[key]; !ok {
			continue
		}

		dest := filepath.Join(outDir, fmt.Sprintf("%s-%s-%s-%s", r.Name, r.Version, p.OS, p.Arch))
		if err := r.fetchPlatform(p.OS, p.Arch, dest); err != nil {
			return fmt.Errorf("fetching %s: %w", key, err)
		}
		if err := os.Chmod(dest, 0o755); err != nil {
			return fmt.Errorf("chmod %s: %w", dest, err)
		}
		binaries[key] = dest
	}

	if len(binaries) == 0 {
		return fmt.Errorf("no platforms have checksums")
	}

	args := []string{"bin", "build", "-n", r.Name}
	for _, t := range r.Tags {
		args = append(args, "-t", fmt.Sprintf("%s/%s:%s", registry, r.Name, t))
	}
	if r.Description != "" {
		args = append(args, "-d", r.Description)
	}
	if r.Source != "" {
		args = append(args, "-s", r.Source)
	}
	if r.Usage != "" {
		args = append(args, "-u", r.Usage)
	}
	for _, p := range platforms {
		key := p.OS + "/" + p.Arch
		if path, ok := binaries[key]; ok {
			args = append(args, fmt.Sprintf("%s:%s", key, path))
		}
	}
	if err := runCmd(ottersBin, args...); err != nil {
		return fmt.Errorf("otters bin build: %w", err)
	}

	if !push {
		return nil
	}

	for _, t := range r.Tags {
		ref := fmt.Sprintf("%s/%s:%s", registry, r.Name, t)
		if err := runCmd(ottersBin, "bin", "push", ref); err != nil {
			return fmt.Errorf("otters bin push %s: %w", ref, err)
		}
	}
	return nil
}

func pushedNote(pushed bool) string {
	if pushed {
		return ", pushed"
	}
	return ""
}

func loadDescriptor(path string) (*Descriptor, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var d Descriptor
	if err := yaml.Unmarshal(b, &d); err != nil {
		return nil, err
	}
	return &d, nil
}

func (d *Descriptor) validate() error {
	if d.Name == "" {
		return errors.New("name is required")
	}
	if len(d.Versions) == 0 {
		return errors.New("versions: must contain at least one entry")
	}
	seenVersion := map[string]bool{}
	seenAlias := map[string]string{} // alias -> version it's claimed by
	for _, v := range d.Versions {
		if v.Version == "" {
			return errors.New("every version entry needs `version`")
		}
		if seenVersion[v.Version] {
			return fmt.Errorf("duplicate version %q", v.Version)
		}
		seenVersion[v.Version] = true

		if len(v.Checksums) == 0 {
			return fmt.Errorf("version %s: checksums map is required", v.Version)
		}

		// `version` itself can't appear as an alias — that's redundant
		// and would cause `otters bin build` to be passed the same -t
		// twice.
		for _, a := range v.Aliases {
			if a == v.Version {
				return fmt.Errorf("version %s: alias %q duplicates the version itself", v.Version, a)
			}
			if prev, ok := seenAlias[a]; ok {
				return fmt.Errorf("alias %q claimed by both versions %s and %s", a, prev, v.Version)
			}
			seenAlias[a] = v.Version
		}
	}
	return nil
}

// resolveVersions merges per-version overrides over the descriptor
// defaults and computes the final tag list per version. If versionRef
// is non-empty, only that version is returned.
func (d *Descriptor) resolveVersions(versionRef string) ([]Resolved, error) {
	// First pass: determine whether any version explicitly claims `latest`
	// so we know whether to add it implicitly to the first entry.
	latestClaimed := false
	for _, v := range d.Versions {
		for _, a := range v.Aliases {
			if a == "latest" {
				latestClaimed = true
			}
		}
	}

	out := make([]Resolved, 0, len(d.Versions))
	for i, v := range d.Versions {
		if versionRef != "" && v.Version != versionRef {
			continue
		}
		r := d.merge(v)
		// First entry implicitly gets `latest` when nobody claims it.
		if i == 0 && !latestClaimed {
			r.Tags = append(r.Tags, "latest")
		}
		// Validate every required field is now present after merge.
		switch {
		case r.URLTemplate == "":
			return nil, fmt.Errorf("version %s: url_template missing (no default, no override)", v.Version)
		case r.Archive == "":
			return nil, fmt.Errorf("version %s: archive missing (no default, no override)", v.Version)
		case r.Archive != "raw" && r.BinaryInArchive == "":
			return nil, fmt.Errorf("version %s: binary_in_archive required for archived downloads", v.Version)
		}
		out = append(out, r)
	}

	if versionRef != "" && len(out) == 0 {
		return nil, fmt.Errorf("version %q not found in descriptor", versionRef)
	}
	return out, nil
}

// merge produces a Resolved by overlaying per-version fields on top of
// the descriptor defaults. Non-empty version fields win; maps replace
// wholesale (no deep merge).
func (d *Descriptor) merge(v VersionEntry) Resolved {
	pick := func(version, def string) string {
		if version != "" {
			return version
		}
		return def
	}
	pickMap := func(version, def map[string]string) map[string]string {
		if version != nil {
			return version
		}
		return def
	}

	tags := []string{v.Version}
	tags = append(tags, v.Aliases...)

	return Resolved{
		Name:            d.Name,
		Description:     pick(v.Description, d.Description),
		Source:          d.Source,
		Usage:           pick(v.Usage, d.Usage),
		Version:         v.Version,
		URLTemplate:     pick(v.URLTemplate, d.URLTemplate),
		Archive:         pick(v.Archive, d.Archive),
		BinaryInArchive: pick(v.BinaryInArchive, d.BinaryInArchive),
		OSAlias:         pickMap(v.OSAlias, d.OSAlias),
		ArchAlias:       pickMap(v.ArchAlias, d.ArchAlias),
		Checksums:       v.Checksums,
		Platforms:       v.Platforms,
		Tags:            tags,
	}
}

func (r *Resolved) fetchPlatform(osName, arch, dest string) error {
	pr, err := r.Platform(osName, arch)
	if err != nil {
		return err
	}

	url, err := r.renderTemplate(pr.URLTemplate, osName, arch, pr.Target)
	if err != nil {
		return fmt.Errorf("render url_template: %w", err)
	}

	tmp, err := os.CreateTemp("", fmt.Sprintf("vendor-%s-%s-%s-%s-*", r.Name, r.Version, osName, arch))
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(tmp.Name()) }()

	if err := download(url, tmp); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("downloading %s: %w", url, err)
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	if err := verifySHA256(tmp.Name(), pr.Checksum); err != nil {
		return fmt.Errorf("checksum mismatch for %s: %w", url, err)
	}

	return r.extractBinary(tmp.Name(), dest, osName, arch, pr)
}

// renderTemplate runs `tmpl` with the standard substitution set —
// {{.Version}} / {{.OS}} / {{.Arch}} / {{.Target}} — applying os_alias
// and arch_alias before substitution. `target` is the per-platform
// string from PlatformEntry.Target (empty for tools that don't need
// it).
//
// Available template funcs:
//
//	trimPrefix "v"   strips a leading "v" — for upstreams whose URL
//	                 form drops the v even though the tag has one
//	                 (gh ships gh_2.92.0_... but tags v2.92.0).
//	trimSuffix "..." mirror of the above.
func (r *Resolved) renderTemplate(tmpl, osName, arch, target string) (string, error) {
	funcs := template.FuncMap{
		"trimPrefix": func(prefix, s string) string { return strings.TrimPrefix(s, prefix) },
		"trimSuffix": func(suffix, s string) string { return strings.TrimSuffix(s, suffix) },
	}
	t, err := template.New("vendor").Funcs(funcs).Parse(tmpl)
	if err != nil {
		return "", err
	}
	if alias, ok := r.OSAlias[osName]; ok {
		osName = alias
	}
	if alias, ok := r.ArchAlias[arch]; ok {
		arch = alias
	}
	var sb strings.Builder
	err = t.Execute(&sb, map[string]string{
		"Version": r.Version,
		"OS":      osName,
		"Arch":    arch,
		"Target":  target,
	})
	return sb.String(), err
}

func download(url string, w io.Writer) error {
	resp, err := http.Get(url) //nolint:gosec // url comes from a trusted descriptor
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %s", resp.Status)
	}

	_, err = io.Copy(w, resp.Body)
	return err
}

func verifySHA256(path, want string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}
	got := hex.EncodeToString(h.Sum(nil))
	if !strings.EqualFold(got, want) {
		return fmt.Errorf("got %s, want %s", got, want)
	}
	return nil
}

func (r *Resolved) extractBinary(archivePath, dest, osName, arch string, pr PlatformResolved) error {
	binPath, err := r.renderTemplate(pr.BinaryInArchive, osName, arch, pr.Target)
	if err != nil {
		return fmt.Errorf("render binary_in_archive: %w", err)
	}
	switch pr.Archive {
	case "raw":
		return copyFile(archivePath, dest)
	case "tar.gz":
		f, err := os.Open(archivePath)
		if err != nil {
			return err
		}
		defer func() { _ = f.Close() }()
		gz, err := gzip.NewReader(f)
		if err != nil {
			return err
		}
		defer func() { _ = gz.Close() }()
		return extractFromTar(tar.NewReader(gz), binPath, dest)
	case "tar":
		f, err := os.Open(archivePath)
		if err != nil {
			return err
		}
		defer func() { _ = f.Close() }()
		return extractFromTar(tar.NewReader(f), binPath, dest)
	case "zip":
		return extractFromZip(archivePath, binPath, dest)
	}
	return fmt.Errorf("unknown archive type %q", pr.Archive)
}

func extractFromTar(tr *tar.Reader, wantPath, dest string) error {
	for {
		h, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return fmt.Errorf("binary %q not found in archive", wantPath)
		}
		if err != nil {
			return err
		}
		if h.Name == wantPath || strings.TrimPrefix(h.Name, "./") == wantPath {
			return writeAll(tr, dest)
		}
	}
}

func extractFromZip(archivePath, wantPath, dest string) error {
	zr, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer func() { _ = zr.Close() }()

	for _, f := range zr.File {
		if f.Name == wantPath {
			rc, err := f.Open()
			if err != nil {
				return err
			}
			defer func() { _ = rc.Close() }()
			return writeAll(rc, dest)
		}
	}
	return fmt.Errorf("binary %q not found in zip archive", wantPath)
}

func writeAll(r io.Reader, dest string) error {
	out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()
	_, err = io.Copy(out, r)
	return err
}

func copyFile(src, dest string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()
	return writeAll(in, dest)
}

func runCmd(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "bintool-vendor: "+format+"\n", args...)
	os.Exit(1)
}
