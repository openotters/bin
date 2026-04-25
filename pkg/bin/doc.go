// Package bin builds, extracts, and inspects OCI bin-tool images —
// the public API underneath the bintool examples and (in Phase 2)
// the openotters daemon's tool-build RPC.
//
// A bin image is a regular OCI image (single-arch manifest or
// multi-arch index) carrying vnd.openotters.bin.* annotations that
// describe a static binary and optional metadata. The image index
// and per-platform manifests advertise spec.BinArtifactType so tool
// images are trivially distinguishable from agent images.
//
//	Bin OCI Image
//	+--------------------------------------------------+
//	| manifest  artifactType = vnd.openotters.bin.v1   |
//	|   annotations:                                   |
//	|     vnd.openotters.bin.name = "jq"               |
//	|     vnd.openotters.bin.path = "/"                |
//	|     vnd.openotters.bin.description = "..."       |
//	|     vnd.openotters.bin.usage = "/USAGE.md"       |
//	|                                                  |
//	|   layers:                                        |
//	|     [0] jq        (binary, title annotation)     |
//	|     [1] USAGE.md  (optional, title annotation)   |
//	+--------------------------------------------------+
//	              |
//	              v
//	  BuildIndex(opts, platforms, dst) -> multi-arch image
//	  Build(opts, src, dst)            -> single-platform image
//	  Inspect(manifest)                -> Info (name, path, …)
//	  ExtractBin(fetcher, manifest, …) -> writes binary to billy.FS
//	  FetchUsage(fetcher, manifest)    -> USAGE.md content as string
package bin
