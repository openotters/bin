# bintool examples

Small Go programs that exercise each stage of a tool image's lifecycle.
Each one is a self-contained `main.go` you can run directly with
`go run` — think of them as runnable documentation for the
`github.com/openotters/bin/pkg/bin` and `github.com/openotters/agentfile/oci`
APIs.

| Example | Purpose |
|---|---|
| [`build`](./build/README.md) | Build a multi-arch OCI tool image from one binary per platform and push it to a registry. |
| [`info`](./info/README.md) | Pull a tool image and print its metadata (bin path, description, `USAGE.md`). |
| [`pull`](./pull/README.md) | Pull a tool image into an in-memory store and summarise what arrived (digest, artifact type, layer count). |
| [`push`](./push/README.md) | Mirror a tool image from one registry ref to another, staging in memory. |
| [`extract`](./extract/README.md) | Pull a tool image and write the binary for the current host OS/arch to a directory. |
| [`validate`](./validate/README.md) | Check that a ref is a well-formed bin-tool image (artifact type, required annotations, binary layer present). Exits non-zero on failure. |

## Running

All examples share the same authentication story as `oras`/Docker —
credentials come from `~/.docker/config.json` via credential helpers.
Use `-plain-http` (or `-plain-http-src`/`-plain-http-dst` for `push`)
for local registries without TLS.

```sh
# Build jq for three platforms and push
GOOS=linux  GOARCH=amd64 CGO_ENABLED=0 go build -o /tmp/jq-linux-amd64  ./some/jq
GOOS=linux  GOARCH=arm64 CGO_ENABLED=0 go build -o /tmp/jq-linux-arm64  ./some/jq
GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -o /tmp/jq-darwin-arm64 ./some/jq
go run ./examples/build -name jq -desc "JSON processor" \
  ghcr.io/openotters/tools/jq:0.1.0 \
  linux/amd64:/tmp/jq-linux-amd64 \
  linux/arm64:/tmp/jq-linux-arm64 \
  darwin/arm64:/tmp/jq-darwin-arm64

# Inspect it
go run ./examples/info    ghcr.io/openotters/tools/jq:latest
go run ./examples/pull    ghcr.io/openotters/tools/jq:latest

# Pull & extract the binary for the current host
go run ./examples/extract ghcr.io/openotters/tools/jq:latest /tmp
/tmp/jq --version

# Check the image is spec-compliant
go run ./examples/validate ghcr.io/openotters/tools/jq:latest
go run ./examples/validate ghcr.io/some/random/image:latest   # → exit 1

# Mirror to another registry
go run ./examples/push \
  ghcr.io/openotters/tools/jq:latest \
  registry.internal/mirror/jq:latest
```

## Library primitives

The examples above are thin wrappers — the real work happens in:

- `bintool/pkg/bin/build.go` — `BuildOptions`, `PlatformBuild`, `BuildIndex`, `Build`.
- `bintool/pkg/bin/info.go` — `Inspect(manifest) Info`.
- `bintool/pkg/bin/extract.go` — `Extract(ctx, fetcher, manifest, fs, dest)`.
- `bintool/pkg/bin/usage.go` — `FetchUsage(ctx, fetcher, manifest)`.
- `agentfile/oci` — `NewRemoteRepository`, `ResolveManifest`, `FetchBlobBytes`, `AgentFetcher`.
- `agentfile/spec/mediatype.go` — `BinArtifactType`, `Annotation*` constants.

These live under `pkg/` (stable public API), so external code can
import them directly: `import bin "github.com/openotters/bin/pkg/bin"`.
