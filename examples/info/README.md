# info

Pull a tool image from a registry and print its metadata: bin path,
description, layer summary, and the body of the `USAGE.md` layer if
present. Demonstrates `bin.Inspect` and `bin.FetchUsage`.

## Usage

```sh
go run ./examples/info/ [-plain-http] <registry-ref>
```

| Flag | Description |
|---|---|
| `-plain-http` | Talk plain HTTP to the registry (local registries). |

## Example

```sh
go run ./examples/info/ ghcr.io/openotters/tools/jq:latest
```

```
name:        jq
path:        /
bin:         /jq
description: JSON processor
usage path:  /USAGE.md
layers:      2
  jq                   application/octet-stream  4990802 bytes  sha256:6c0a1b3d…
  /USAGE.md            text/markdown  251 bytes  sha256:9f4e5a8b…

--- USAGE.md ---
…
```

## Library API

- [`bin.Inspect`](https://pkg.go.dev/github.com/openotters/bin/pkg/bin#Inspect)
- [`bin.FetchUsage`](https://pkg.go.dev/github.com/openotters/bin/pkg/bin#FetchUsage)
