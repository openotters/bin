# Vendored tools

Tools we don't reimplement in Go — we package the upstream binary.

Each `<name>.yaml` in this directory is a descriptor read by
`cmd/bintool-vendor`. It says where to download the upstream binary
for each `os/arch`, what SHA256 the download must hash to, and what
metadata to bake into the resulting OCI BIN image.

Drive the pipeline with `task tools:vendor:publish TOOL=<name>`.

## Descriptor schema

```yaml
name: yaegi                 # tool name (becomes /<name> inside the image)
version: v0.16.1            # upstream version — used in URL substitution
description: "..."          # one-line description, shown to LLMs as tool intent
source: https://...         # upstream repo — stamped as image source

# Go template. Variables: .Version, .OS (linux|darwin), .Arch (amd64|arm64).
url_template: "https://github.com/{owner}/{repo}/releases/download/{{.Version}}/{{.Version}}_{{.OS}}_{{.Arch}}.tar.gz"

archive: tar.gz             # tar.gz | tar | zip | raw
binary_in_archive: yaegi    # path of the binary inside the archive (omit for raw)

checksums:
  darwin/arm64: <sha256>
  darwin/amd64: <sha256>
  linux/arm64:  <sha256>
  linux/amd64:  <sha256>

usage: |
  Free-form markdown baked into the image as USAGE.md so the runtime
  (and any LLM) can introspect the tool's expected I/O contract.
```

## Why this exists

The original `tools:publish` pipeline only knows how to compile Go source
under `cmd/tools/<name>`. That works for tools we can cleanly express as a
Go program (`jq` via gojq, `sh` via u-root). It doesn't work for tools
that have no usable Go-library form: kubectl, helm, ffmpeg, pandoc,
crane, ast-grep, duckdb, etc.

The vendored pipeline pulls upstream binaries instead. One descriptor
per tool, one repackaging step. Image content is byte-identical to what
the upstream project ships; we just wrap it in OCI BIN annotations so
the openotters runtime can dispatch it.
