# bintool

[![Go Reference](https://pkg.go.dev/badge/github.com/openotters/bin.svg)](https://pkg.go.dev/github.com/openotters/bin)
[![Go Report Card](https://goreportcard.com/badge/github.com/openotters/bin)](https://goreportcard.com/report/github.com/openotters/bin)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE.md)

Build, inspect, and extract OCI bin images — static binaries packaged as OCI artifacts with
`vnd.openotters.bin.*` annotations.

<!-- TOC -->
* [bintool](#bintool)
  * [What is a bin image?](#what-is-a-bin-image)
  * [Tools](#tools)
  * [Building & publishing](#building--publishing)
  * [Library API](#library-api)
  * [License](#license)
<!-- TOC -->

## What is a bin image?

A bin image is a regular OCI image that carries annotations describing a static binary:

```
manifest
  annotations:
    vnd.openotters.bin.name: jq
    vnd.openotters.bin.path: /
    vnd.openotters.bin.description: JSON processor
    vnd.openotters.bin.usage: /USAGE.md
  layers:
    [0] jq        (binary)
    [1] USAGE.md  (optional usage documentation)
```

Any OCI image can adopt these annotations. The annotation contract is defined in the
[Agentfile specification](https://github.com/openotters/agentfile/blob/main/AGENTFILE-v1.0.0.md#binary-oci-image-structure).

## Tools

48 ready-to-use tool binaries for AI agents, published at `ghcr.io/openotters/tools/{name}:latest`:

`base64` `basename` `cat` `chmod` `cp` `date` `dirname` `echo` `false` `find` `grep` `gzip`
`head` `hostname` `id` `jina` `jq` `ln` `ls` `mkdir` `mktemp` `more` `mv` `ping` `printenv`
`pwd` `readlink` `realpath` `rm` `rmdir` `seq` `sh` `shasum` `sleep` `sort` `tail` `tee`
`time` `touch` `tr` `true` `uname` `uniq` `wc` `wget` `which` `xargs` `yes`

`jina` fetches URL content as clean markdown; `sh` is a minimal POSIX shell for
agents that need to pipe or redirect between tools.

Each tool is a plain CLI — argv in, stdout out. When the openotters
runtime invokes a BIN tool on behalf of an agent, it shell-splits the
LLM's input string and execs the binary directly; stdout is the
response, a non-zero exit code surfaces as an error. There is no
JSON envelope.

Each tool image embeds a `USAGE.md` layer describing its expected
argv and flags.

## Building & publishing

Cross-build a tool for `darwin/{arm64,amd64}` + `linux/{arm64,amd64}` and
push it as a multi-arch bin image with `task`:

```sh
# A single tool
task tools:publish TOOL=wget

# Every tool under cmd/tools/
task tools:publish

# Override the registry / otters CLI
REGISTRY=registry.internal/mirror task tools:publish TOOL=jq
OTTERS="go run ../openotters/cmd/otters" task tools:publish TOOL=jq
```

The task wraps `otters bin build` + `otters bin push`; `otters` must be
on `$PATH` (or set `OTTERS=…`).

For programmatic use, the `examples/` directory has runnable Go
programs for each lifecycle stage (`build`, `info`, `pull`, `push`,
`extract`, `validate`) — see [`examples/README.md`](./examples/README.md).

## Library API

The bin-image primitives are in the importable package
[`github.com/openotters/bin/pkg/bin`](https://pkg.go.dev/github.com/openotters/bin/pkg/bin):

- `Build` / `BuildIndex` — assemble multi-arch bin images.
- `Inspect` — read `vnd.openotters.bin.*` metadata off a manifest.
- `Extract` / `FetchUsage` — pull a binary or its `USAGE.md` from
  a fetcher.

OCI transport (resolve, fetch blobs, push) lives in
[`agentfile/oci`](https://pkg.go.dev/github.com/openotters/agentfile/oci);
artifact-type and annotation constants live in
[`agentfile/spec`](https://pkg.go.dev/github.com/openotters/agentfile/spec).

## License

See [LICENSE](LICENSE.md).
