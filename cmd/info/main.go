// Toolinfo pulls a tool image from a registry and prints its metadata (bin path, description, usage).
//
// Usage:
//
//	go run ./cmd/info/ [-plain-http] <registry-ref>
//
// Example:
//
//	go run ./cmd/info/ ghcr.io/openotters/tools/wget:latest
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/openotters/bin/internal"
	oci2 "github.com/openotters/bin/internal/oci"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/memory"
)

func main() {
	plainHTTP := flag.Bool("plain-http", false, "use plain HTTP instead of HTTPS")
	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: toolinfo [-plain-http] <registry-ref>")
		os.Exit(1)
	}

	ref := args[0]

	var opts []oci2.RemoteRepositoryOption
	if *plainHTTP {
		opts = append(opts, oci2.WithPlainHTTP)
	}

	repo, err := oci2.NewRemoteRepository(ref, opts...)
	if err != nil {
		fatal(err)
	}

	tag := repo.Reference.Reference
	if tag == "" {
		tag = "latest"
	}

	store := memory.New()

	_, err = oras.Copy(context.Background(), repo, tag, store, tag, oras.DefaultCopyOptions)
	if err != nil {
		fatal(err)
	}

	manifest, err := oci2.ResolveManifest(context.Background(), repo, must(repo.Resolve(context.Background(), tag)))
	if err != nil {
		fatal(err)
	}

	info := internal.Inspect(*manifest)

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
		usage, usageErr := internal.FetchUsage(context.Background(), store, *manifest)
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
