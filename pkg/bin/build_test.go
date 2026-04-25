package bin_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"testing"

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

func resolveManifest(t *testing.T, store *memory.Store, desc v1.Descriptor) (*v1.Manifest, error) {
	t.Helper()

	rc, err := store.Fetch(context.Background(), desc)
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		return nil, err
	}

	var manifest v1.Manifest
	if unmarshalErr := json.Unmarshal(data, &manifest); unmarshalErr != nil {
		return nil, unmarshalErr
	}

	return &manifest, nil
}
