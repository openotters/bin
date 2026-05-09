# Vendored tools

Tools we don't reimplement in Go — we package the upstream binary.

Each `<name>.yaml` in this directory is a descriptor read by
`cmd/bintool-vendor`. It describes one tool, possibly across multiple
versions, and resolves to a multi-arch OCI BIN image per version.

Drive the pipeline with:

```sh
task tools:vendor:publish TOOL=<name>                # all versions of one tool
task tools:vendor:publish TOOL=<name> VERSION=x.y.z  # one version of one tool
task tools:vendor:publish                            # all versions of every tool
```

## Descriptor schema

```yaml
name: <tool-name>            # required, top-level only
description: "..."           # default; per-version override allowed
source: https://...          # default; per-version override allowed
usage: |                     # default; per-version override allowed
  multi-line markdown content baked into the image as USAGE.md

# ── Defaults inherited by every version unless overridden ──
url_template: "https://.../{{.Version}}/{{.OS}}-{{.Arch}}.tar.gz"
archive: tar.gz              # tar.gz | tar | zip | raw
binary_in_archive: <path>    # required unless archive=raw
os_alias:                    # only used when rendering URLs (checksums always use Go names)
  darwin: macos
arch_alias:
  amd64: x86_64

# ── Version list. First entry implicitly gets `latest` ──
versions:
  - version: "1.8.1"
    aliases: ["1.8", "1"]
    checksums:
      darwin/arm64: <sha256>
      darwin/amd64: <sha256>
      linux/arm64:  <sha256>
      linux/amd64:  <sha256>

  - version: "1.7.1"
    aliases: ["1.7"]
    checksums:
      ...

  - version: "1.6.0"
    # Override: pre-1.7 lived under stedolan/jq with a different layout.
    url_template: "https://github.com/stedolan/jq/releases/download/jq-{{.Version}}/jq-{{.OS}}-{{.Arch}}.tar.gz"
    archive: tar.gz
    binary_in_archive: jq
    os_alias:
      darwin: osx
    checksums:
      ...
```

## Merge & tag rules

**Per-version override.** Any non-empty field on a version entry
replaces the descriptor default for that field. Maps (`os_alias`,
`arch_alias`, `checksums`) replace wholesale — there's no deep merge.
If you want a one-key delta on `os_alias`, spell out the whole map on
the version entry. Easier to reason about; harder to surprise.

**Required on a version.** `version` (string), `checksums` (map keyed
on Go-style `<os>/<arch>` for all four supported platforms). Everything
else inherits.

**Tags pushed for one version.** Always `<name>:<version>`, plus
everything in `aliases: []`. Plus `latest` for the *first* entry in
`versions:` if and only if no version explicitly claims `latest` in
its aliases.

That last rule means:

- New tool, only one version → automatic `latest`.
- Multi-version tool, top is current → automatic `latest` on top.
- Multi-version tool but you want `latest` pinned to an older
  known-stable release → list `aliases: ["latest"]` on that version
  and the implicit add stops.

**Validation.** Two version entries can't claim the same alias
(would silently move the tag); a version can't list its own version
string as an alias (would pass `-t` twice to `otters bin build`); every
version needs all four platform checksums.

## The runtime contract this maps to

The openotters runtime invokes BIN tools with two structured fields:

- `args` — list of positional arguments passed as argv
- `stdin` — content piped to the binary's stdin (optional)

So vendoring works for any upstream CLI whose meaningful input is
expressible as argv, stdin, or both. That covers nearly all real
tools: kubectl / helm / crane (argv only), yaegi / pandoc / jq with
`-` (argv + stdin), ffmpeg (`-i pipe:0`), and so on.

The few cases vendoring still can't handle: tools that need
interactive multi-stream prompts or persistent state between calls.
Those need a real Go wrapper under `cmd/tools/<name>/`.
