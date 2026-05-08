package bin

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
	"strconv"
	"time"

	"github.com/go-git/go-billy/v6"
	"github.com/opencontainers/go-digest"
	"github.com/opencontainers/image-spec/specs-go"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content"
	"oras.land/oras-go/v2/errdef"

	"github.com/openotters/agentfile/spec"
)

type BuildOptions struct {
	Name        string
	BinPath     string // path to read the binary from src filesystem
	Path        string // vnd.openotters.bin.path (default "/")
	Description string
	Usage       string
	// Source is the URL of the upstream repo. When set, gets
	// stamped as the OCI standard `org.opencontainers.image.source`
	// annotation on every produced manifest + index. ghcr.io
	// uses this annotation to auto-link the package to a GitHub
	// repository and inherit its visibility — the difference
	// between "public package" and "manually flip private→public
	// in the UI for every tool".
	Source string
}

// Build is a single-platform convenience that wraps BuildIndex
// with one entry — the host OS / Arch and the binary at
// opts.BinPath. Output is always a multi-arch index (with one
// child manifest), so the on-disk shape matches BuildIndex's and
// callers can branch on `mediaType=index` without checking how
// the producer was invoked.
//
// Useful for tests and quick local builds where you don't care
// about cross-compilation; production callers should use
// BuildIndex with an explicit platform set so the result runs on
// every host openotters expects to dispatch to (the system
// executor needs the host platform; the docker executor needs
// linux/<host-arch> at minimum).
func Build(ctx context.Context, opts BuildOptions, src billy.Filesystem, dst oras.Target) (*digest.Digest, error) {
	platform := PlatformBuild{
		OS:      runtime.GOOS,
		Arch:    runtime.GOARCH,
		BinPath: opts.BinPath,
		Src:     src,
	}

	return BuildIndex(ctx, opts, []PlatformBuild{platform}, dst)
}

type PlatformBuild struct {
	OS      string
	Arch    string
	BinPath string
	Src     billy.Filesystem
}

func BuildIndex(
	ctx context.Context, opts BuildOptions, platforms []PlatformBuild, dst oras.Target,
) (*digest.Digest, error) {
	var manifests []v1.Descriptor

	for _, p := range platforms {
		platformTag := fmt.Sprintf("latest-%s-%s", p.OS, p.Arch)

		platOpts := opts
		if p.BinPath != "" {
			platOpts.BinPath = p.BinPath
		}

		_, err := buildPlatform(ctx, platOpts, p.Src, dst, platformTag, p.OS, p.Arch)
		if err != nil {
			return nil, fmt.Errorf("%s/%s: %w", p.OS, p.Arch, err)
		}

		desc, err := dst.Resolve(ctx, platformTag)
		if err != nil {
			return nil, fmt.Errorf("resolving %s: %w", platformTag, err)
		}

		desc.Platform = &v1.Platform{OS: p.OS, Architecture: p.Arch}
		manifests = append(manifests, desc)
	}

	// Mirror the per-platform manifest's bin annotations onto the
	// index so consumers can Inspect either shape and get the same
	// info — Build (single-platform → index of one) and BuildIndex
	// (multi-platform) become indistinguishable downstream.
	indexAnnotations := map[string]string{
		spec.AnnotationBinName: opts.Name,
	}

	binPath := opts.Path
	if binPath == "" {
		binPath = spec.DefaultBinPath
	}
	indexAnnotations[spec.AnnotationBinPath] = binPath

	if opts.Description != "" {
		indexAnnotations[spec.AnnotationBinDescription] = opts.Description
	}

	if opts.Usage != "" {
		indexAnnotations[spec.AnnotationBinUsage] = spec.DefaultUsagePath
	}

	if opts.Description != "" {
		indexAnnotations[v1.AnnotationDescription] = opts.Description
	}

	if opts.Source != "" {
		indexAnnotations[v1.AnnotationSource] = opts.Source
	}

	index := v1.Index{
		Versioned:    specs.Versioned{SchemaVersion: 2},
		MediaType:    v1.MediaTypeImageIndex,
		ArtifactType: spec.BinArtifactType,
		Manifests:    manifests,
		Annotations:  indexAnnotations,
	}

	indexData, err := json.Marshal(index)
	if err != nil {
		return nil, fmt.Errorf("marshaling index: %w", err)
	}

	indexDesc := v1.Descriptor{
		MediaType: v1.MediaTypeImageIndex,
		Digest:    digestOf(indexData),
		Size:      int64(len(indexData)),
	}

	if err = dst.Push(ctx, indexDesc, bytes.NewReader(indexData)); err != nil && !alreadyExists(err) {
		return nil, fmt.Errorf("pushing index: %w", err)
	}

	if err = dst.Tag(ctx, indexDesc, "latest"); err != nil {
		return nil, fmt.Errorf("tagging index: %w", err)
	}

	d := indexDesc.Digest
	return &d, nil
}

// buildPlatform produces a real OCI image for one platform: layer is
// a gzipped tar containing the binary at /<name>, config is a valid
// v1.Image with rootfs.diff_ids + Architecture/OS so Docker engines
// can run / image-mount / inspect the result. The bin annotations
// (vnd.openotters.bin.*) live alongside the standard image fields so
// custom tooling that reads them still works.
//
//nolint:funlen // full Docker-image assembly inline; splitting fragments the data flow
func buildPlatform(
	ctx context.Context, opts BuildOptions, src billy.Filesystem, dst oras.Target, tag, osName, archName string,
) (*digest.Digest, error) {
	binData, err := readFile(src, opts.BinPath)
	if err != nil {
		return nil, fmt.Errorf("reading binary %s: %w", opts.BinPath, err)
	}

	// Wrap the binary in a tar.gz at /<name>. diffID is the digest of
	// the *uncompressed* tar (per the OCI spec); the layer descriptor
	// digest below is over the gzipped bytes.
	tarBytes, err := buildTarLayer(opts.Name, binData)
	if err != nil {
		return nil, fmt.Errorf("building tar layer: %w", err)
	}

	diffID := digestOf(tarBytes)

	gzBytes, err := gzipBytes(tarBytes)
	if err != nil {
		return nil, fmt.Errorf("gzipping tar layer: %w", err)
	}

	binPath := opts.Path
	if binPath == "" {
		binPath = spec.DefaultBinPath
	}

	annotations := map[string]string{
		spec.AnnotationBinName: opts.Name,
		spec.AnnotationBinPath: binPath,
	}

	if opts.Description != "" {
		annotations[spec.AnnotationBinDescription] = opts.Description
		annotations[v1.AnnotationDescription] = opts.Description
	}

	if opts.Source != "" {
		annotations[v1.AnnotationSource] = opts.Source
	}

	binLayerDesc, err := pushBlob(ctx, dst, v1.MediaTypeImageLayerGzip, gzBytes, map[string]string{
		// Keep AnnotationTitle so the Puller's blob-by-title fast
		// path still finds the binary blob without unpacking the tar.
		v1.AnnotationTitle: opts.Name,
	})
	if err != nil {
		return nil, fmt.Errorf("pushing binary layer: %w", err)
	}

	layers := []v1.Descriptor{binLayerDesc}

	if opts.Usage != "" {
		usageName := spec.DefaultUsagePath
		annotations[spec.AnnotationBinUsage] = usageName

		usageDesc, usageErr := pushBlob(ctx, dst, spec.Markdown, []byte(opts.Usage), map[string]string{
			v1.AnnotationTitle: usageName,
		})
		if usageErr != nil {
			return nil, fmt.Errorf("pushing usage: %w", usageErr)
		}

		layers = append(layers, usageDesc)
	}

	// A valid OCI image config: Architecture/OS for platform
	// selection, Config.Entrypoint so `docker run` resolves to the
	// embedded binary, RootFS.DiffIDs listing the uncompressed-tar
	// digest of every rootfs-bearing layer (only the bin layer
	// here — the usage doc is metadata, not a rootfs layer, so it's
	// excluded).
	configLabels := map[string]string{}
	if opts.Source != "" {
		// ghcr.io reads `org.opencontainers.image.source` from
		// the image config Labels (Docker convention) and uses
		// it to auto-link the package to the named GitHub repo,
		// inheriting the repo's visibility on first push. The
		// same key as a manifest annotation is informational
		// only — the linking happens via the LABEL, not the
		// annotation.
		configLabels["org.opencontainers.image.source"] = opts.Source
	}
	if opts.Description != "" {
		configLabels["org.opencontainers.image.description"] = opts.Description
	}

	imgConfig := v1.Image{
		Created:  ptrTime(buildTimestamp()),
		Platform: v1.Platform{Architecture: archName, OS: osName},
		Config: v1.ImageConfig{
			Entrypoint: []string{"/" + opts.Name},
			Cmd:        nil,
			WorkingDir: "/",
			Labels:     configLabels,
		},
		RootFS: v1.RootFS{
			Type:    "layers",
			DiffIDs: []digest.Digest{diffID},
		},
	}

	configData, err := json.Marshal(imgConfig)
	if err != nil {
		return nil, fmt.Errorf("marshaling image config: %w", err)
	}

	configDesc, err := pushBlob(ctx, dst, v1.MediaTypeImageConfig, configData, nil)
	if err != nil {
		return nil, fmt.Errorf("pushing config: %w", err)
	}

	manifest := v1.Manifest{
		Versioned:    specs.Versioned{SchemaVersion: 2},
		MediaType:    v1.MediaTypeImageManifest,
		ArtifactType: spec.BinArtifactType,
		Config:       configDesc,
		Layers:       layers,
		Annotations:  annotations,
	}

	manifestData, err := json.Marshal(manifest)
	if err != nil {
		return nil, fmt.Errorf("marshaling manifest: %w", err)
	}

	manifestDesc := v1.Descriptor{
		MediaType: v1.MediaTypeImageManifest,
		Digest:    digestOf(manifestData),
		Size:      int64(len(manifestData)),
	}

	if err = dst.Push(ctx, manifestDesc, bytes.NewReader(manifestData)); err != nil && !alreadyExists(err) {
		return nil, fmt.Errorf("pushing manifest: %w", err)
	}

	if err = dst.Tag(ctx, manifestDesc, tag); err != nil {
		return nil, fmt.Errorf("tagging manifest: %w", err)
	}

	d := manifestDesc.Digest
	return &d, nil
}

// buildTarLayer writes binData as a single executable file at /<name>
// inside a tar archive, returning the raw (uncompressed) bytes.
// Mode 0755 so the file is executable inside the container; UID/GID
// 0 so a non-root user that bind-mounts this in a multi-stage build
// can still read it (Docker images are conventionally root-owned).
//
// The tar entry's mtime is held at unix-0 so the layer digest only
// depends on the binary content — two builds of the same binary
// produce identical layer blobs, which lets the embedded oras
// registry dedupe them. The image config's Created field carries
// the actual build time (see buildTimestamp), so `docker images`
// shows a sensible age.
func buildTarLayer(name string, binData []byte) ([]byte, error) {
	var buf bytes.Buffer

	tw := tar.NewWriter(&buf)

	hdr := &tar.Header{
		Name:    name,
		Mode:    0o755,
		Size:    int64(len(binData)),
		ModTime: time.Unix(0, 0).UTC(),
		Uid:     0,
		Gid:     0,
		Format:  tar.FormatPAX,
	}

	if err := tw.WriteHeader(hdr); err != nil {
		return nil, err
	}

	if _, err := tw.Write(binData); err != nil {
		return nil, err
	}

	if err := tw.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// buildTimestamp returns the time stamped into the OCI image config's
// Created field. Honours SOURCE_DATE_EPOCH (a widely-supported
// reproducible-builds convention) when set, falling back to wall-clock
// time. Reproducible-builds CI pipelines export SOURCE_DATE_EPOCH so
// the image config — and therefore the manifest digest — is stable;
// interactive `otters bin build` runs land at the current time so
// `docker images` shows a sensible age column instead of "56 years
// ago" (which earlier builds produced from a hard-coded unix-0).
func buildTimestamp() time.Time {
	if raw := os.Getenv("SOURCE_DATE_EPOCH"); raw != "" {
		if secs, err := strconv.ParseInt(raw, 10, 64); err == nil {
			return time.Unix(secs, 0).UTC()
		}
	}

	return time.Now().UTC()
}

func gzipBytes(data []byte) ([]byte, error) {
	var buf bytes.Buffer

	gz := gzip.NewWriter(&buf)

	if _, err := gz.Write(data); err != nil {
		return nil, err
	}

	if err := gz.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func ptrTime(t time.Time) *time.Time { return &t }

func readFile(fs billy.Filesystem, path string) ([]byte, error) {
	f, err := fs.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening %s: %w", path, err)
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	return data, nil
}

func pushBlob(
	ctx context.Context, dst content.Pusher, mediaType string, data []byte, annotations map[string]string,
) (v1.Descriptor, error) {
	desc := v1.Descriptor{
		MediaType:   mediaType,
		Digest:      digestOf(data),
		Size:        int64(len(data)),
		Annotations: annotations,
	}

	if err := dst.Push(ctx, desc, bytes.NewReader(data)); err != nil && !alreadyExists(err) {
		return v1.Descriptor{}, fmt.Errorf("pushing blob: %w", err)
	}

	return desc, nil
}

func digestOf(data []byte) digest.Digest {
	h := sha256.Sum256(data)
	return digest.NewDigestFromBytes(digest.SHA256, h[:])
}

// digestOfHex is unused but retained to document the shape of a
// hex-prefixed sha256 — referenced when wiring future signing.
//
//nolint:unused // documentation helper retained intentionally
func digestOfHex(data []byte) string {
	h := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(h[:])
}

func alreadyExists(err error) bool {
	return errors.Is(err, errdef.ErrAlreadyExists)
}
