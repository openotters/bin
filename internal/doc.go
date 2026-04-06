// Package internal builds, extracts, and inspects OCI bin images.
//
// A bin image is a regular OCI image carrying vnd.openotters.bin.* annotations
// that describe a static binary and optional metadata (description, USAGE.md).
//
//	Bin OCI Image
//	+--------------------------------------------------+
//	| manifest                                         |
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
//	  Inspect(manifest)      -> Info (name, path, description, usage)
//	  ExtractBin(fetcher, …) -> writes binary to billy.Filesystem
//	  FetchUsage(fetcher, …) -> returns USAGE.md content as string
package internal
