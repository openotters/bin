// bintool-vendor reads a vendored-tool descriptor (vendor/<name>.yaml),
// downloads the upstream binary for darwin/linux × amd64/arm64,
// verifies SHA256 checksums, extracts each binary, and shells out to
// `otters bin build` (and optionally `otters bin push`) to package the
// result as an OCI BIN image.
//
// Usage:
//
//	bintool-vendor -descriptor vendor/yaegi.yaml -registry ghcr.io/openotters/tools \
//	    [-out /tmp/otters-bin] [-otters otters] [-push]
//
// Drives the `tools:vendor:publish` Taskfile target.
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

type Descriptor struct {
	Name            string            `yaml:"name"`
	Version         string            `yaml:"version"`
	Description     string            `yaml:"description"`
	Source          string            `yaml:"source"`
	URLTemplate     string            `yaml:"url_template"`
	Archive         string            `yaml:"archive"` // "tar.gz" | "tar" | "zip" | "raw"
	BinaryInArchive string            `yaml:"binary_in_archive"`
	Checksums       map[string]string `yaml:"checksums"` // key: "<os>/<arch>" (Go convention, e.g. darwin/arm64)
	Usage           string            `yaml:"usage"`

	// Optional remappings used during URL substitution when the
	// upstream uses different os/arch tokens than Go's GOOS/GOARCH.
	// Example: jq publishes "macos" instead of "darwin", so the
	// jq.yaml descriptor sets `os_alias: { darwin: macos }`.
	// The `os/arch` keys in `checksums` always use Go conventions —
	// aliases only affect URL templating.
	OSAlias   map[string]string `yaml:"os_alias"`
	ArchAlias map[string]string `yaml:"arch_alias"`
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

	// Download + extract per-platform binaries.
	binaries := make(map[string]string, len(platforms)) // "os/arch" -> path
	for _, p := range platforms {
		key := p.OS + "/" + p.Arch
		want, ok := d.Checksums[key]
		if !ok {
			fatal("descriptor missing checksum for %s", key)
		}

		dest := filepath.Join(*outDir, fmt.Sprintf("%s-%s-%s", d.Name, p.OS, p.Arch))
		if err := d.fetchPlatform(p.OS, p.Arch, want, dest); err != nil {
			fatal("fetching %s: %v", key, err)
		}
		if err := os.Chmod(dest, 0o755); err != nil {
			fatal("chmod %s: %v", dest, err)
		}
		binaries[key] = dest
	}

	// Compose the otters bin build invocation.
	tag := fmt.Sprintf("%s/%s:latest", *registry, d.Name)
	args := []string{"bin", "build", "-n", d.Name, "-t", tag}
	if d.Description != "" {
		args = append(args, "-d", d.Description)
	}
	if d.Source != "" {
		args = append(args, "-s", d.Source)
	}
	if d.Usage != "" {
		args = append(args, "-u", d.Usage)
	}
	for _, p := range platforms {
		key := p.OS + "/" + p.Arch
		args = append(args, fmt.Sprintf("%s:%s", key, binaries[key]))
	}
	if err := runCmd(*ottersBin, args...); err != nil {
		fatal("otters bin build: %v", err)
	}

	if *push {
		if err := runCmd(*ottersBin, "bin", "push", tag); err != nil {
			fatal("otters bin push: %v", err)
		}
	}

	fmt.Printf("ok — %s built (%d platforms)%s\n", tag, len(platforms), pushedNote(*push))
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
	if d.Version == "" {
		return errors.New("version is required")
	}
	if d.URLTemplate == "" {
		return errors.New("url_template is required")
	}
	switch d.Archive {
	case "tar.gz", "tar", "zip", "raw":
	default:
		return fmt.Errorf("archive must be one of tar.gz|tar|zip|raw (got %q)", d.Archive)
	}
	if d.Archive != "raw" && d.BinaryInArchive == "" {
		return errors.New("binary_in_archive is required for archived downloads")
	}
	if len(d.Checksums) == 0 {
		return errors.New("checksums map is required")
	}
	return nil
}

func (d *Descriptor) fetchPlatform(osName, arch, wantSHA, dest string) error {
	url, err := d.renderURL(osName, arch)
	if err != nil {
		return err
	}

	tmp, err := os.CreateTemp("", fmt.Sprintf("vendor-%s-%s-%s-*", d.Name, osName, arch))
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

	if err := verifySHA256(tmp.Name(), wantSHA); err != nil {
		return fmt.Errorf("checksum mismatch for %s: %w", url, err)
	}

	return d.extractBinary(tmp.Name(), dest)
}

func (d *Descriptor) renderURL(osName, arch string) (string, error) {
	t, err := template.New("url").Parse(d.URLTemplate)
	if err != nil {
		return "", err
	}
	if alias, ok := d.OSAlias[osName]; ok {
		osName = alias
	}
	if alias, ok := d.ArchAlias[arch]; ok {
		arch = alias
	}
	var sb strings.Builder
	err = t.Execute(&sb, map[string]string{
		"Version": d.Version,
		"OS":      osName,
		"Arch":    arch,
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

func (d *Descriptor) extractBinary(archivePath, dest string) error {
	switch d.Archive {
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
		return extractFromTar(tar.NewReader(gz), d.BinaryInArchive, dest)
	case "tar":
		f, err := os.Open(archivePath)
		if err != nil {
			return err
		}
		defer func() { _ = f.Close() }()
		return extractFromTar(tar.NewReader(f), d.BinaryInArchive, dest)
	case "zip":
		return extractFromZip(archivePath, d.BinaryInArchive, dest)
	}
	return fmt.Errorf("unknown archive type %q", d.Archive)
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
