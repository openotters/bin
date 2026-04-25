// Push mirrors a tool image from one registry to another. It pulls
// the source ref into an in-memory store, then copies everything to
// the destination ref. The local memory store is the "staging" area
// — no file system touched.
//
// Usage:
//
//	go run ./examples/push/ [-plain-http-src] [-plain-http-dst] <src-ref> <dst-ref>
//
// Example:
//
//	go run ./examples/push/ ghcr.io/openotters/tools/jq:latest registry.internal/mirror/jq:latest
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
)

func main() {
	plainHTTPSrc := flag.Bool("plain-http-src", false, "use plain HTTP for the source registry")
	plainHTTPDst := flag.Bool("plain-http-dst", false, "use plain HTTP for the destination registry")
	flag.Parse()

	args := flag.Args()
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: push [-plain-http-src] [-plain-http-dst] <src-ref> <dst-ref>")
		os.Exit(1)
	}

	srcRef := spec.ParseReference(args[0])
	dstRef := spec.ParseReference(args[1])

	ctx := context.Background()

	src, err := openRepo(srcRef, *plainHTTPSrc)
	if err != nil {
		fatal(fmt.Errorf("source: %w", err))
	}

	dst, err := openRepo(dstRef, *plainHTTPDst)
	if err != nil {
		fatal(fmt.Errorf("destination: %w", err))
	}

	// Stage the image in memory with the SRC tag, then re-tag in place
	// so the pushed tag matches the destination.
	stage := memory.New()

	desc, err := oras.Copy(ctx, src, srcRef.Tag, stage, srcRef.Tag, oras.DefaultCopyOptions)
	if err != nil {
		fatal(fmt.Errorf("pulling %s: %w", srcRef, err))
	}

	if err = stage.Tag(ctx, desc, dstRef.Tag); err != nil {
		fatal(fmt.Errorf("retag: %w", err))
	}

	if _, err = oras.Copy(ctx, stage, dstRef.Tag, dst, dstRef.Tag, oras.DefaultCopyOptions); err != nil {
		fatal(fmt.Errorf("pushing %s: %w", dstRef, err))
	}

	fmt.Fprintf(os.Stdout, "mirrored %s → %s (%s)\n", srcRef, dstRef, desc.Digest)
}

func openRepo(ref spec.Reference, plainHTTP bool) (oras.Target, error) {
	var opts []oci.RemoteRepositoryOption
	if plainHTTP {
		opts = append(opts, oci.WithPlainHTTP)
	}

	return oci.NewRemoteRepository(ref, opts...)
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
