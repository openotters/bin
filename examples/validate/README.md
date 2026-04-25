# validate

Check that an OCI ref points at a well-formed openotters bin-tool
image: the correct `artifactType`, the required `vnd.openotters.bin.*`
annotations, and a binary layer matching the declared name. Exits 0
on success, 1 with a readable report otherwise.

Use it to gate publishing pipelines, or to verify that a third-party
image actually conforms to the bin-tool contract before running it.

## Usage

```sh
go run ./examples/validate/ [-plain-http] <registry-ref>
```

| Flag | Description |
|---|---|
| `-plain-http` | Talk plain HTTP to the registry (local registries). |

## Example

```sh
go run ./examples/validate/ ghcr.io/openotters/tools/jq:latest
#   ✓ artifactType = application/vnd.openotters.bin.v1
#   ✓ vnd.openotters.bin.name = jq
#   ✓ vnd.openotters.bin.path = /
#   ✓ vnd.openotters.bin.description = JSON processor
#   ✓ binary layer present: jq

go run ./examples/validate/ ghcr.io/some/random/image:latest   # → exit 1
```

## Library API

- [`oci.ResolveManifest`](https://pkg.go.dev/github.com/openotters/agentfile/oci#ResolveManifest)
- [`spec.BinArtifactType`](https://pkg.go.dev/github.com/openotters/agentfile/spec#BinArtifactType)
  and `spec.AnnotationBin*` constants.
