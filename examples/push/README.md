# push

Mirror a tool image from one registry to another. Pulls the source
ref into an in-memory store (the "staging" area), retags it for the
destination, then copies the image to the destination registry. No
file system touched.

## Usage

```sh
go run ./examples/push/ [-plain-http-src] [-plain-http-dst] <src-ref> <dst-ref>
```

| Flag | Description |
|---|---|
| `-plain-http-src` | Talk plain HTTP to the **source** registry. |
| `-plain-http-dst` | Talk plain HTTP to the **destination** registry. |

The two flags exist independently so a public→private mirror can pull
over HTTPS and push over HTTP (or vice-versa).

## Example

```sh
go run ./examples/push/ \
  ghcr.io/openotters/tools/jq:latest \
  registry.internal/mirror/jq:latest
```

## Library API

- [`oras.Copy`](https://pkg.go.dev/oras.land/oras-go/v2#Copy) +
  [`memory.New`](https://pkg.go.dev/oras.land/oras-go/v2/content/memory#New)
  for staging
- [`oci.NewRemoteRepository`](https://pkg.go.dev/github.com/openotters/agentfile/oci#NewRemoteRepository)
