# pull

Download a tool image from an OCI registry into an in-memory store
and print the resolved digest, artifact type, and metadata. Useful as
a starting point for programs that want to work with tool images
locally (inspect, re-push, extract, etc.) without re-hitting the
registry.

## Usage

```sh
go run ./examples/pull/ [-plain-http] <registry-ref>
```

| Flag | Description |
|---|---|
| `-plain-http` | Talk plain HTTP to the registry (local registries). |

## Example

```sh
go run ./examples/pull/ ghcr.io/openotters/tools/jq:latest
```

```
pulled ghcr.io/openotters/tools/jq:latest
  digest: sha256:1c2a3f4b…
  size:   1923 bytes
  type:   application/vnd.openotters.bin.v1
  name:   jq
  bin:    /jq
  layers: 2
```

## Library API

- [`oras.Copy`](https://pkg.go.dev/oras.land/oras-go/v2#Copy) into
  [`memory.New`](https://pkg.go.dev/oras.land/oras-go/v2/content/memory#New)
- [`oci.ResolveManifest`](https://pkg.go.dev/github.com/openotters/agentfile/oci#ResolveManifest)
- [`bin.Inspect`](https://pkg.go.dev/github.com/openotters/bin/pkg/bin#Inspect)
