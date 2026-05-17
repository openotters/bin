package bin_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/go-git/go-billy/v6"
	"github.com/go-git/go-billy/v6/memfs"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/openotters/agentfile/spec"
	"github.com/openotters/bin/pkg/bin"
	"oras.land/oras-go/v2/content/memory"
)

func TestBuild_Minimal(t *testing.T) {
	t.Parallel()

	src := memfs.New()
	f, _ := src.Create("jq")
	_, _ = f.Write([]byte("fake-binary"))
	_ = f.Close()

	store := memory.New()

	digest, err := bin.Build(context.Background(), bin.BuildOptions{
		Name:    "jq",
		BinPath: "jq",
	}, src, store)
	if err != nil {
		t.Fatalf("build error: %v", err)
	}

	if digest == nil {
		t.Fatal("nil digest")
	}
}

func TestBuild_WithDescriptionAndUsage(t *testing.T) {
	t.Parallel()

	src := memfs.New()
	f, _ := src.Create("jq")
	_, _ = f.Write([]byte("fake-binary"))
	_ = f.Close()

	store := memory.New()

	_, err := bin.Build(context.Background(), bin.BuildOptions{
		Name:        "jq",
		BinPath:     "jq",
		Description: "Extract fields from JSON",
		Usage:       "First line is the jq expression.\nRest is JSON input.",
	}, src, store)
	if err != nil {
		t.Fatalf("build error: %v", err)
	}

	desc, err := store.Resolve(context.Background(), "latest")
	if err != nil {
		t.Fatalf("resolve error: %v", err)
	}

	manifest, err := resolveManifest(t, store, desc)
	if err != nil {
		t.Fatalf("manifest error: %v", err)
	}

	info := bin.Inspect(*manifest)

	if info.Name != "jq" {
		t.Errorf("name = %q, want jq", info.Name)
	}

	if info.Description != "Extract fields from JSON" {
		t.Errorf("description = %q", info.Description)
	}

	if info.UsagePath != spec.DefaultUsagePath {
		t.Errorf("usage path = %q, want %s", info.UsagePath, spec.DefaultUsagePath)
	}
}

func TestBuildExtract_Roundtrip(t *testing.T) {
	t.Parallel()

	src := memfs.New()
	f, _ := src.Create("jq")
	_, _ = f.Write([]byte("fake-jq-binary"))
	_ = f.Close()

	store := memory.New()

	_, err := bin.Build(context.Background(), bin.BuildOptions{
		Name:        "jq",
		BinPath:     "jq",
		Description: "JSON processor",
		Usage:       "First line is the expression.",
	}, src, store)
	if err != nil {
		t.Fatalf("build error: %v", err)
	}

	desc, err := store.Resolve(context.Background(), "latest")
	if err != nil {
		t.Fatalf("resolve error: %v", err)
	}

	manifest, err := resolveManifest(t, store, desc)
	if err != nil {
		t.Fatalf("manifest error: %v", err)
	}

	dst := memfs.New()
	if extractErr := bin.Extract(context.Background(), store, *manifest, dst, "usr/bin/jq"); extractErr != nil {
		t.Fatalf("extract error: %v", extractErr)
	}

	extracted, err := dst.Open("usr/bin/jq")
	if err != nil {
		t.Fatalf("open error: %v", err)
	}

	buf := make([]byte, 100)
	n, _ := extracted.Read(buf)
	_ = extracted.Close()

	if string(buf[:n]) != "fake-jq-binary" {
		t.Errorf("binary content = %q", string(buf[:n]))
	}

	usage, err := bin.FetchUsage(context.Background(), store, *manifest)
	if err != nil {
		t.Fatalf("usage error: %v", err)
	}

	if usage != "First line is the expression." {
		t.Errorf("usage = %q", usage)
	}
}

func TestBuildIndex_MultiPlatform(t *testing.T) {
	t.Parallel()

	mkSrc := func(content string) billy.Filesystem {
		fs := memfs.New()
		f, _ := fs.Create("jq")
		_, _ = f.Write([]byte(content))
		_ = f.Close()
		return fs
	}

	store := memory.New()
	platforms := []bin.PlatformBuild{
		{OS: "linux", Arch: "amd64", Src: mkSrc("linux-amd64-bytes")},
		{OS: "darwin", Arch: "arm64", Src: mkSrc("darwin-arm64-bytes")},
	}

	idxDigest, err := bin.BuildIndex(context.Background(), bin.BuildOptions{
		Name:    "jq",
		BinPath: "jq",
	}, platforms, store)
	if err != nil {
		t.Fatalf("BuildIndex: %v", err)
	}

	if idxDigest == nil {
		t.Fatal("nil index digest")
	}

	desc, err := store.Resolve(context.Background(), "latest")
	if err != nil {
		t.Fatalf("resolving latest: %v", err)
	}

	if desc.MediaType != v1.MediaTypeImageIndex {
		t.Fatalf("MediaType = %q, want image index", desc.MediaType)
	}

	rc, err := store.Fetch(context.Background(), desc)
	if err != nil {
		t.Fatalf("fetch index: %v", err)
	}
	defer rc.Close()

	data, _ := io.ReadAll(rc)

	var index v1.Index
	if unmarshalErr := json.Unmarshal(data, &index); unmarshalErr != nil {
		t.Fatalf("unmarshal index: %v", unmarshalErr)
	}

	if got := len(index.Manifests); got != 2 {
		t.Fatalf("manifests = %d, want 2", got)
	}

	for i, want := range []string{"linux/amd64", "darwin/arm64"} {
		got := fmt.Sprintf("%s/%s", index.Manifests[i].Platform.OS, index.Manifests[i].Platform.Architecture)
		if got != want {
			t.Errorf("manifest[%d] platform = %q, want %q", i, got, want)
		}
	}

	// Re-pushing the same platform set must succeed (alreadyExists path).
	if _, rePushErr := bin.BuildIndex(context.Background(), bin.BuildOptions{
		Name:    "jq",
		BinPath: "jq",
	}, platforms, store); rePushErr != nil {
		t.Fatalf("BuildIndex (re-push): %v", rePushErr)
	}
}

// TestBuild_OCIAnnotations covers the OCI image-spec annotation
// surface. Every key with a non-empty BuildOptions field must land
// on both the index and the per-platform manifest; unset fields
// must NOT produce empty annotations (cleaner for `otters bin
// inspect` output and matches the spec note that empty values are
// allowed but absence is cleaner). The created annotation is
// always present and parses as RFC 3339.
func TestBuild_OCIAnnotations(t *testing.T) {
	t.Parallel()

	src := memfs.New()
	f, _ := src.Create("jq")
	_, _ = f.Write([]byte("fake-binary"))
	_ = f.Close()

	store := memory.New()

	_, err := bin.Build(context.Background(), bin.BuildOptions{
		Name:          "jq",
		BinPath:       "jq",
		Description:   "JSON processor",
		Source:        "https://github.com/jqlang/jq",
		Version:       "1.7.1",
		Revision:      "abc123def456",
		Licenses:      "MIT",
		Vendor:        "jqlang",
		Authors:       "stedolan@example.com",
		URL:           "https://jqlang.org",
		Documentation: "https://jqlang.org/manual",
	}, src, store)
	if err != nil {
		t.Fatalf("build error: %v", err)
	}

	desc, err := store.Resolve(context.Background(), "latest")
	if err != nil {
		t.Fatalf("resolve error: %v", err)
	}

	indexBytes, err := fetchBytes(store, desc)
	if err != nil {
		t.Fatalf("fetch index: %v", err)
	}

	var index v1.Index
	if uerr := json.Unmarshal(indexBytes, &index); uerr != nil {
		t.Fatalf("unmarshal index: %v", uerr)
	}

	manifest, err := resolveManifest(t, store, desc)
	if err != nil {
		t.Fatalf("manifest error: %v", err)
	}

	want := map[string]string{
		v1.AnnotationDescription:   "JSON processor",
		v1.AnnotationSource:        "https://github.com/jqlang/jq",
		v1.AnnotationVersion:       "1.7.1",
		v1.AnnotationRevision:      "abc123def456",
		v1.AnnotationLicenses:      "MIT",
		v1.AnnotationVendor:        "jqlang",
		v1.AnnotationAuthors:       "stedolan@example.com",
		v1.AnnotationURL:           "https://jqlang.org",
		v1.AnnotationDocumentation: "https://jqlang.org/manual",

		// io.openotters.bin.name carries the binary filename for
		// the puller. Distinct from image.title (human-readable
		// display label) which is intentionally not auto-stamped
		// from Name — the two concepts can diverge.
		spec.AnnotationBinName: "jq",
	}

	// Index + per-platform manifest must carry the same OCI keys.
	// Anything readable on the multi-arch entry must also be readable
	// on the platform-specific entry so a single-platform pull and
	// a multi-arch pull look the same from the consumer's side.
	for _, target := range []struct {
		name string
		ann  map[string]string
	}{
		{"index", index.Annotations},
		{"manifest", manifest.Annotations},
	} {
		for k, v := range want {
			if got := target.ann[k]; got != v {
				t.Errorf("%s annotation %s = %q, want %q", target.name, k, got, v)
			}
		}

		if got := target.ann[v1.AnnotationCreated]; got == "" {
			t.Errorf("%s missing %s", target.name, v1.AnnotationCreated)
		} else if _, parseErr := time.Parse(time.RFC3339, got); parseErr != nil {
			t.Errorf("%s %s = %q, not RFC 3339: %v", target.name, v1.AnnotationCreated, got, parseErr)
		}

		// io.openotters.bin.path is always stamped (defaults to "/").
		if got := target.ann[spec.AnnotationBinPath]; got != spec.DefaultBinPath {
			t.Errorf("%s %s = %q, want %q", target.name, spec.AnnotationBinPath, got, spec.DefaultBinPath)
		}
	}
}

// TestBuild_OptInAnnotationsAbsentWhenUnset is the negative half of
// the surface check: when the caller passes empty strings for the
// opt-in fields, those annotations MUST be absent from the produced
// manifest entirely (rather than written as ""). Cleaner inspect
// output and matches the spec hint about presence vs absence.
func TestBuild_OptInAnnotationsAbsentWhenUnset(t *testing.T) {
	t.Parallel()

	src := memfs.New()
	f, _ := src.Create("jq")
	_, _ = f.Write([]byte("fake-binary"))
	_ = f.Close()

	store := memory.New()

	_, err := bin.Build(context.Background(), bin.BuildOptions{
		Name:    "jq",
		BinPath: "jq",
	}, src, store)
	if err != nil {
		t.Fatalf("build error: %v", err)
	}

	desc, err := store.Resolve(context.Background(), "latest")
	if err != nil {
		t.Fatalf("resolve error: %v", err)
	}

	manifest, err := resolveManifest(t, store, desc)
	if err != nil {
		t.Fatalf("manifest error: %v", err)
	}

	mustBeAbsent := []string{
		v1.AnnotationTitle, // not auto-stamped from Name; intentional
		v1.AnnotationDescription,
		v1.AnnotationSource,
		v1.AnnotationVersion,
		v1.AnnotationRevision,
		v1.AnnotationLicenses,
		v1.AnnotationVendor,
		v1.AnnotationAuthors,
		v1.AnnotationURL,
		v1.AnnotationDocumentation,
	}
	for _, key := range mustBeAbsent {
		if _, present := manifest.Annotations[key]; present {
			t.Errorf("expected %s absent when unset, got value %q",
				key, manifest.Annotations[key])
		}
	}

	// bin.name (binary filename for the puller) and created are
	// always expected — name from opts.Name, created auto-stamped.
	if manifest.Annotations[spec.AnnotationBinName] != "jq" {
		t.Errorf("%s = %q, want jq",
			spec.AnnotationBinName, manifest.Annotations[spec.AnnotationBinName])
	}
	if _, ok := manifest.Annotations[v1.AnnotationCreated]; !ok {
		t.Errorf("missing %s", v1.AnnotationCreated)
	}
}

// resolveManifest fetches the bytes at desc and decodes them as a
// v1.Manifest. When desc points at an index (Build now always
// produces one — Build wraps BuildIndex with a single host-
// platform entry), the helper follows the first child entry to
// the actual manifest blob. Tests rely on this so they can call
// Build / BuildIndex interchangeably and inspect the same shape.
func resolveManifest(t *testing.T, store *memory.Store, desc v1.Descriptor) (*v1.Manifest, error) {
	t.Helper()

	data, err := fetchBytes(store, desc)
	if err != nil {
		return nil, err
	}

	if isIndexMediaType(desc.MediaType) {
		var index v1.Index
		if unmarshalErr := json.Unmarshal(data, &index); unmarshalErr != nil {
			return nil, fmt.Errorf("decode index: %w", unmarshalErr)
		}
		if len(index.Manifests) == 0 {
			return nil, fmt.Errorf("index has no manifests")
		}
		data, err = fetchBytes(store, index.Manifests[0])
		if err != nil {
			return nil, err
		}
	}

	var manifest v1.Manifest
	if unmarshalErr := json.Unmarshal(data, &manifest); unmarshalErr != nil {
		return nil, unmarshalErr
	}

	return &manifest, nil
}

func fetchBytes(store *memory.Store, desc v1.Descriptor) ([]byte, error) {
	rc, err := store.Fetch(context.Background(), desc)
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	return io.ReadAll(rc)
}

func isIndexMediaType(mt string) bool {
	return mt == v1.MediaTypeImageIndex
}
