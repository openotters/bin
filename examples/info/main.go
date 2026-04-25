// Info pulls a tool image from a registry and prints its metadata
// (bin path, description, usage text). Rich view — includes the
// USAGE.md body if declared.
//
// Usage:
//
//	go run ./examples/info/ [-plain-http] <registry-ref>
//
// Example:
//
//	go run ./examples/info/ ghcr.io/openotters/tools/wget:latest
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
		fmt.Fprintln(os.Stderr, "usage: info [-plain-http] <registry-ref>")
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

	tag := ref.Tag

	store := memory.New()

	_, err = oras.Copy(context.Background(), repo, tag, store, tag, oras.DefaultCopyOptions)
	if err != nil {
		fatal(err)
	}

	manifest, err := oci.ResolveManifest(context.Background(), repo, must(repo.Resolve(context.Background(), tag)))
	if err != nil {
		fatal(err)
	}

	info := bin.Inspect(*manifest)

	fmt.Fprintf(os.Stdout, "name:        %s\n", info.Name)
	fmt.Fprintf(os.Stdout, "path:        %s\n", info.Path)
	fmt.Fprintf(os.Stdout, "bin:         %s\n", info.BinPath())
	fmt.Fprintf(os.Stdout, "description: %s\n", info.Description)
	fmt.Fprintf(os.Stdout, "usage path:  %s\n", info.UsagePath)
	fmt.Fprintf(os.Stdout, "layers:      %d\n", len(info.Layers))

	for _, l := range info.Layers {
		title := l.Title
		if title == "" {
			title = "(untitled)"
		}

		fmt.Fprintf(os.Stdout, "  %-20s %s  %d bytes  %s\n", title, l.MediaType, l.Size, l.Digest[:19])
	}

	if info.UsagePath != "" {
		usage, usageErr := bin.FetchUsage(context.Background(), store, *manifest)
		if usageErr != nil {
			fatal(usageErr)
		}

		fmt.Fprintf(os.Stdout, "\n--- USAGE.md ---\n%s\n", usage)
	}
}

func must[T any](v T, err error) T {
	if err != nil {
		fatal(err)
	}

	return v
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
