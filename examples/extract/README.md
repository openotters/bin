# extract

Pull a tool image and write the binary for the current host's
OS+arch into `<dest-dir>/<bin-name>` with executable permissions.
Useful for installing a tool locally or priming a sandbox directory.
Demonstrates `bin.Extract` against a `billy.Filesystem` destination.

## Usage

```sh
go run ./examples/extract/ [-plain-http] <registry-ref> <dest-dir>
```

| Flag | Description |
|---|---|
| `-plain-http` | Talk plain HTTP to the registry (local registries). |

## Example

```sh
go run ./examples/extract/ ghcr.io/openotters/tools/jq:latest /tmp
/tmp/jq --help
```

## Library API

- [`bin.Extract`](https://pkg.go.dev/github.com/openotters/bin/pkg/bin#Extract)
- [`oci.NewRemoteRepository`](https://pkg.go.dev/github.com/openotters/agentfile/oci#NewRemoteRepository),
  [`oci.ResolveManifest`](https://pkg.go.dev/github.com/openotters/agentfile/oci#ResolveManifest)
