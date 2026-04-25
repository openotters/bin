// Extract pulls a tool image and writes the binary for the current
// host's OS+arch into <dest-dir>/<bin-name> with executable permissions.
// Useful for installing a tool locally or priming a sandbox directory.
//
// Usage:
//
//	go run ./examples/extract/ [-plain-http] <registry-ref> <dest-dir>
//
// Example:
//
//	go run ./examples/extract/ ghcr.io/openotters/tools/jq:latest /tmp
//	/tmp/jq --help
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-git/go-billy/v6/osfs"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/memory"

	"github.com/openotters/agentfile/oci"
	"github.com/openotters/agentfile/spec"
	"github.com/openotters/bin/pkg/bin"
)

func main() {
	plainHTTP := flag.Bool("plain-http", false, "use plain HTTP instead of HTTPS")
	flag.Parse()

	args := flag.Args()
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: extract [-plain-http] <registry-ref> <dest-dir>")
		os.Exit(1)
	}

	ref := spec.ParseReference(args[0])

	destDir, err := filepath.Abs(args[1])
	if err != nil {
		fatal(err)
	}

	if statErr := os.MkdirAll(destDir, 0o755); statErr != nil {
		fatal(fmt.Errorf("creating dest: %w", statErr))
	}

	var opts []oci.RemoteRepositoryOption
	if *plainHTTP {
		opts = append(opts, oci.WithPlainHTTP)
	}

	repo, err := oci.NewRemoteRepository(ref, opts...)
	if err != nil {
		fatal(err)
	}

	ctx := context.Background()

	store := memory.New()

	desc, err := oras.Copy(ctx, repo, ref.Tag, store, ref.Tag, oras.DefaultCopyOptions)
	if err != nil {
		fatal(fmt.Errorf("copying: %w", err))
	}

	// ResolveManifest walks indexes and picks the runtime GOOS/GOARCH
	// submanifest automatically.
	manifest, err := oci.ResolveManifest(ctx, store, desc)
	if err != nil {
		fatal(fmt.Errorf("resolving manifest: %w", err))
	}

	info := bin.Inspect(*manifest)
	if info.Name == "" {
		fatal(fmt.Errorf("manifest has no %s annotation; not a bin-tool image", spec.AnnotationBinName))
	}

	fs := osfs.New(destDir)
	if err = bin.Extract(ctx, store, *manifest, fs, info.Name); err != nil {
		fatal(fmt.Errorf("extracting %s: %w", info.Name, err))
	}

	fmt.Fprintf(os.Stdout, "extracted %s → %s\n", info.Name, filepath.Join(destDir, info.Name))
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
