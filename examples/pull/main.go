// Pull downloads a tool image from an OCI registry into an in-memory
// store and prints the resolved digest + metadata. Useful as a
// starting point for programs that want to work with tool images
// locally (inspect, re-push, extract, etc.) without re-hitting the
// registry.
//
// Usage:
//
//	go run ./examples/pull/ [-plain-http] <registry-ref>
//
// Example:
//
//	go run ./examples/pull/ ghcr.io/openotters/tools/jq:latest
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

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
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: pull [-plain-http] <registry-ref>")
		os.Exit(1)
	}

	ref := spec.ParseReference(args[0])

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

	manifest, err := oci.ResolveManifest(ctx, store, desc)
	if err != nil {
		fatal(fmt.Errorf("resolving manifest: %w", err))
	}

	info := bin.Inspect(*manifest)

	fmt.Fprintf(os.Stdout, "pulled %s\n", ref)
	fmt.Fprintf(os.Stdout, "  digest: %s\n", desc.Digest)
	fmt.Fprintf(os.Stdout, "  size:   %d bytes\n", desc.Size)
	fmt.Fprintf(os.Stdout, "  type:   %s\n", manifest.ArtifactType)
	fmt.Fprintf(os.Stdout, "  name:   %s\n", info.Name)
	fmt.Fprintf(os.Stdout, "  bin:    %s\n", info.BinPath())
	fmt.Fprintf(os.Stdout, "  layers: %d\n", len(info.Layers))
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
