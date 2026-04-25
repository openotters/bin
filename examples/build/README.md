# build

Build per-platform OCI tool images and assemble a multi-arch image
index, then push the result to a registry. Demonstrates
`bin.BuildIndex` — the higher-level entry point that produces the
shape `otters` consumes when an agent declares a tool dependency.

Authentication uses Docker credential helpers
(`~/.docker/config.json`).

## Usage

```sh
go run ./examples/build/ [-plain-http] -name <name> [-desc <desc>] [-usage <text>] \
  <ref> <os/arch:path> [<os/arch:path> ...]
```

| Flag | Description |
|---|---|
| `-name` | Tool name (required) — embedded in `vnd.openotters.bin.name`. |
| `-desc` | One-line description — embedded in `vnd.openotters.bin.description`. |
| `-usage` | Usage text — pushed as a `USAGE.md` layer. |
| `-plain-http` | Talk plain HTTP to the destination (local registries). |

Each `<os/arch:path>` argument is one platform's pre-built binary on
the host filesystem. The binary's basename becomes the in-image binary
name.

## Example

```sh
go run ./examples/build/ -name jq -desc "JSON processor" \
  ghcr.io/openotters/tools/jq:0.1.0 \
  linux/amd64:bin/jq-linux-amd64 \
  linux/arm64:bin/jq-linux-arm64 \
  darwin/arm64:bin/jq-darwin-arm64
```

## Library API

- [`bin.BuildIndex`](https://pkg.go.dev/github.com/openotters/bin/pkg/bin#BuildIndex)
- [`bin.BuildOptions`](https://pkg.go.dev/github.com/openotters/bin/pkg/bin#BuildOptions),
  [`bin.PlatformBuild`](https://pkg.go.dev/github.com/openotters/bin/pkg/bin#PlatformBuild)
