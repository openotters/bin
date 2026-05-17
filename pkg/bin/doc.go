// Package bin builds, extracts, and inspects OCI bin-tool images —
// the public API underneath the bintool examples and the openotters
// daemon's tool-build RPC.
//
// A bin image is a regular OCI image (single-arch manifest or
// multi-arch index). OCI image-spec predefined annotations carry
// the standard metadata (title, description, source, version,
// revision, licenses, vendor, …); io.openotters.bin.* carries the
// runtime-specific extras the OCI spec doesn't cover (mount path,
// usage doc path). The image index and per-platform manifests
// advertise spec.BinArtifactType
// (application/vnd.openotters.bin.v1, per RFC 6838 vendor media
// type conventions — distinct from the annotation namespace) so
// tool images are trivially distinguishable from agent images.
//
//	Bin OCI Image
//	+----------------------------------------------------+
//	| manifest  artifactType = vnd.openotters.bin.v1     |
//	|   annotations:                                     |
//	|     org.opencontainers.image.description = "..."   |
//	|     org.opencontainers.image.source      = "..."   |
//	|     org.opencontainers.image.version     = "1.7.1" |
//	|     org.opencontainers.image.created     = RFC3339 |
//	|     io.openotters.bin.name  = "jq"                 |
//	|     io.openotters.bin.path  = "/"                  |
//	|     io.openotters.bin.usage = "/USAGE.md"          |
//	|                                                    |
//	|   layers:                                          |
//	|     [0] jq        (binary, title annotation)       |
//	|     [1] USAGE.md  (optional, title annotation)     |
//	+----------------------------------------------------+
//	              |
//	              v
//	  BuildIndex(opts, platforms, dst) -> multi-arch image
//	  Build(opts, src, dst)            -> single-platform image
//	  Inspect(manifest)                -> Info (name, path, …)
//	  ExtractBin(fetcher, manifest, …) -> writes binary to billy.FS
//	  FetchUsage(fetcher, manifest)    -> USAGE.md content as string
package bin
