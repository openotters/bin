package bin_test

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/go-git/go-billy/v6/memfs"
	"github.com/opencontainers/go-digest"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/openotters/bin/pkg/bin"
)

// stubFetcher implements content.Fetcher backed by an in-memory map
// keyed by digest. Lets tests hand-craft layers without touching a
// real OCI store.
type stubFetcher struct {
	blobs map[string][]byte
}

func (s *stubFetcher) Fetch(_ context.Context, target v1.Descriptor) (io.ReadCloser, error) {
	data, ok := s.blobs[target.Digest.String()]
	if !ok {
		return nil, errors.New("not found")
	}

	return io.NopCloser(bytes.NewReader(data)), nil
}

func tarLayer(t *testing.T, files map[string]string) []byte {
	t.Helper()

	var buf bytes.Buffer

	tw := tar.NewWriter(&buf)
	for name, content := range files {
		if err := tw.WriteHeader(&tar.Header{
			Name:     name,
			Mode:     0o755,
			Size:     int64(len(content)),
			Typeflag: tar.TypeReg,
		}); err != nil {
			t.Fatalf("tar header: %v", err)
		}

		if _, err := tw.Write([]byte(content)); err != nil {
			t.Fatalf("tar write: %v", err)
		}
	}

	if err := tw.Close(); err != nil {
		t.Fatalf("tar close: %v", err)
	}

	return buf.Bytes()
}

func gzipBytes(t *testing.T, data []byte) []byte {
	t.Helper()

	var buf bytes.Buffer

	gw := gzip.NewWriter(&buf)
	if _, err := gw.Write(data); err != nil {
		t.Fatalf("gzip write: %v", err)
	}

	if err := gw.Close(); err != nil {
		t.Fatalf("gzip close: %v", err)
	}

	return buf.Bytes()
}

func sha256Digest(data []byte) digest.Digest {
	sum := sha256.Sum256(data)
	return digest.Digest("sha256:" + hex.EncodeToString(sum[:]))
}

func TestExtract_TarLayer(t *testing.T) {
	t.Parallel()

	tarBytes := tarLayer(t, map[string]string{
		"usr/bin/jq": "fake-jq-bytes",
		"README":     "ignore me",
	})

	d := sha256Digest(tarBytes)

	manifest := v1.Manifest{
		Annotations: map[string]string{
			"vnd.openotters.bin.name": "jq",
			"vnd.openotters.bin.path": "/usr/bin",
		},
		Layers: []v1.Descriptor{{
			MediaType: "application/vnd.oci.image.layer.v1.tar",
			Digest:    d,
			Size:      int64(len(tarBytes)),
		}},
	}

	fetcher := &stubFetcher{blobs: map[string][]byte{d.String(): tarBytes}}
	dst := memfs.New()

	if err := bin.Extract(context.Background(), fetcher, manifest, dst, "jq"); err != nil {
		t.Fatalf("Extract: %v", err)
	}

	f, err := dst.Open("jq")
	if err != nil {
		t.Fatalf("open extracted: %v", err)
	}
	defer f.Close()

	got, _ := io.ReadAll(f)
	if string(got) != "fake-jq-bytes" {
		t.Fatalf("extracted content = %q, want fake-jq-bytes", string(got))
	}
}

func TestExtract_TarGzipLayer(t *testing.T) {
	t.Parallel()

	tarBytes := tarLayer(t, map[string]string{"usr/bin/jq": "compressed-jq"})
	gzBytes := gzipBytes(t, tarBytes)

	d := sha256Digest(gzBytes)

	manifest := v1.Manifest{
		Annotations: map[string]string{
			"vnd.openotters.bin.name": "jq",
			"vnd.openotters.bin.path": "/usr/bin",
		},
		Layers: []v1.Descriptor{{
			MediaType: "application/vnd.oci.image.layer.v1.tar+gzip",
			Digest:    d,
			Size:      int64(len(gzBytes)),
		}},
	}

	fetcher := &stubFetcher{blobs: map[string][]byte{d.String(): gzBytes}}
	dst := memfs.New()

	if err := bin.Extract(context.Background(), fetcher, manifest, dst, "jq"); err != nil {
		t.Fatalf("Extract: %v", err)
	}

	f, err := dst.Open("jq")
	if err != nil {
		t.Fatalf("open extracted: %v", err)
	}
	defer f.Close()

	got, _ := io.ReadAll(f)
	if string(got) != "compressed-jq" {
		t.Fatalf("extracted = %q, want compressed-jq", string(got))
	}
}

func TestExtract_NoMatchingLayer(t *testing.T) {
	t.Parallel()

	tarBytes := tarLayer(t, map[string]string{"some/other/file": "not-the-binary"})
	d := sha256Digest(tarBytes)

	manifest := v1.Manifest{
		Annotations: map[string]string{
			"vnd.openotters.bin.name": "jq",
			"vnd.openotters.bin.path": "/usr/bin",
		},
		Layers: []v1.Descriptor{{
			MediaType: "application/vnd.oci.image.layer.v1.tar",
			Digest:    d,
			Size:      int64(len(tarBytes)),
		}},
	}

	fetcher := &stubFetcher{blobs: map[string][]byte{d.String(): tarBytes}}
	dst := memfs.New()

	err := bin.Extract(context.Background(), fetcher, manifest, dst, "jq")
	if err == nil {
		t.Fatal("Extract returned nil, want error for missing binary")
	}

	if !strings.Contains(err.Error(), "no binary") {
		t.Fatalf("Extract error = %v, want one mentioning missing binary", err)
	}
}
