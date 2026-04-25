// Validate checks that an OCI ref points at a well-formed openotters
// bin-tool image: the correct artifactType, the required
// vnd.openotters.bin.* annotations, and a binary layer matching the
// declared name. Exits 0 on success, 1 with a readable report
// otherwise.
//
// Usage:
//
//	go run ./examples/validate/ [-plain-http] <registry-ref>
//
// Example:
//
//	go run ./examples/validate/ ghcr.io/openotters/tools/jq:latest
//	# OK
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/memory"

	"github.com/openotters/agentfile/oci"
	"github.com/openotters/agentfile/spec"
)

func main() {
	plainHTTP := flag.Bool("plain-http", false, "use plain HTTP instead of HTTPS")
	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: validate [-plain-http] <registry-ref>")
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
		fatal(fmt.Errorf("copying %s: %w", ref, err))
	}

	report := validate(ctx, store, desc)

	for _, line := range report.messages {
		fmt.Fprintln(os.Stdout, line)
	}

	if report.failed {
		os.Exit(1)
	}
}

type checkReport struct {
	failed   bool
	messages []string
}

func (r *checkReport) ok(msg string) { r.messages = append(r.messages, "  ✓ "+msg) }
func (r *checkReport) fail(msg string) {
	r.messages = append(r.messages, "  ✗ "+msg)
	r.failed = true
}

func validate(ctx context.Context, store oras.Target, root v1.Descriptor) checkReport {
	r := checkReport{}

	// Top descriptor should be an index for multi-arch tools, but
	// single-arch manifests are allowed too. If it's an index, walk
	// each submanifest; otherwise check the manifest directly.
	topManifest, err := oci.ResolveManifest(ctx, store, root)
	if err != nil {
		r.fail(fmt.Sprintf("resolve root: %v", err))

		return r
	}

	if topManifest.ArtifactType == spec.BinArtifactType {
		r.ok("artifactType = " + spec.BinArtifactType)
	} else {
		r.fail(fmt.Sprintf("artifactType = %q, want %q", topManifest.ArtifactType, spec.BinArtifactType))
	}

	checkAnnotations(&r, topManifest.Annotations)
	checkBinaryLayer(&r, *topManifest)

	return r
}

func checkAnnotations(r *checkReport, ann map[string]string) {
	name := ann[spec.AnnotationBinName]
	if name == "" {
		r.fail("missing annotation " + spec.AnnotationBinName)
	} else {
		r.ok(spec.AnnotationBinName + " = " + name)
	}

	path := ann[spec.AnnotationBinPath]
	if path == "" {
		r.ok(spec.AnnotationBinPath + " = " + spec.DefaultBinPath + " (default)")
	} else {
		r.ok(spec.AnnotationBinPath + " = " + path)
	}

	if desc := ann[spec.AnnotationBinDescription]; desc != "" {
		r.ok(spec.AnnotationBinDescription + " = " + desc)
	}

	if usage := ann[spec.AnnotationBinUsage]; usage != "" {
		r.ok(spec.AnnotationBinUsage + " = " + usage)
	}
}

func checkBinaryLayer(r *checkReport, manifest v1.Manifest) {
	name := manifest.Annotations[spec.AnnotationBinName]
	if name == "" {
		return
	}

	for _, layer := range manifest.Layers {
		title := layer.Annotations[v1.AnnotationTitle]
		if title == name || filepath.Base(title) == name {
			r.ok("binary layer present: " + title)

			return
		}
	}

	r.fail(fmt.Sprintf("no layer titled %q found", name))
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
