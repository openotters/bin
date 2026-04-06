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
  * [Commands](#commands)
  * [Building tools](#building-tools)
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
[Agentfile specification](https://github.com/openotters/agentfile/blob/main/specs/AGENTFILE-v0.0.1.md#binary-oci-image-structure).

## Tools

46 ready-to-use tool binaries for AI agents, published at `ghcr.io/openotters/tools/{name}:latest`:

`base64` `basename` `cat` `chmod` `cp` `date` `dirname` `echo` `false` `find` `grep` `gzip`
`head` `hostname` `id` `jq` `ln` `ls` `mkdir` `mktemp` `more` `mv` `ping` `printenv` `pwd`
`readlink` `realpath` `rm` `rmdir` `seq` `shasum` `sleep` `sort` `tail` `tee` `time` `touch`
`tr` `true` `uname` `uniq` `wc` `wget` `which` `xargs` `yes`

Each tool follows the same JSON protocol:

```
stdin:  {"input": "args here"}
stdout: {"output": "result here"}
```

Each tool image embeds a `USAGE.md` layer describing its input format and flags.

## Commands

### build

Build a multi-arch bin image and push to a registry:

```sh
go run ./cmd/build/ -name jq -desc "JSON processor" -usage "$(cat cmd/tools/jq/USAGE.md)" \
  ghcr.io/openotters/tools/jq:latest \
  linux/amd64:bin/jq-linux-amd64 \
  linux/arm64:bin/jq-linux-arm64 \
  darwin/amd64:bin/jq-darwin-amd64 \
  darwin/arm64:bin/jq-darwin-arm64
```

### info

Inspect a bin image from a registry:

```sh
go run ./cmd/info/ ghcr.io/openotters/tools/jq:latest
```

```
name:        jq
path:        /
bin:         /jq
description: JSON processor
usage path:  /USAGE.md
layers:      2
  jq                   application/octet-stream  4990802 bytes
  /USAGE.md            text/markdown  251 bytes
```

## Building tools

```sh
# Build a single tool for the current platform
CGO_ENABLED=0 go build -o bin/wget ./cmd/tools/wget/

# Cross-compile and push as multi-arch bin image
for os_arch in linux/amd64 linux/arm64 darwin/amd64 darwin/arm64; do
  GOOS=${os_arch%/*} GOARCH=${os_arch#*/} CGO_ENABLED=0 \
    go build -o bin/wget-${os_arch%/*}-${os_arch#*/} ./cmd/tools/wget/
done

go run ./cmd/build/ -name wget -desc "Fetch URL content" -usage "$(cat cmd/tools/wget/USAGE.md)" \
  ghcr.io/openotters/tools/wget:latest \
  linux/amd64:bin/wget-linux-amd64 \
  linux/arm64:bin/wget-linux-arm64 \
  darwin/amd64:bin/wget-darwin-amd64 \
  darwin/arm64:bin/wget-darwin-arm64
```

## License

See [LICENSE](LICENSE.md).
