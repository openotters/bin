// sh is a tiny POSIX shell BIN backed by mvdan.cc/sh/v3/interp.
// The reason it exists is that openotters' runtime execs each BIN
// tool as a single process — no kernel shell between the LLM and
// the binary — so pipelines (`cmd1 | cmd2 | cmd3`) are impossible
// unless one tool *owns* the pipeline itself. `sh -c "<script>"`
// is that tool.
//
// Why mvdan's shell rather than busybox:
//   - pure Go, cross-compiles to every platform our BIN images
//     target (darwin/arm64, darwin/amd64, linux/amd64, linux/arm64)
//     with zero C toolchain.
//   - embeddable, predictable, no /proc assumptions, no glibc.
//   - good POSIX coverage for the "pipe commands together" case,
//     which is 99% of why an agent would want a shell.
//
// Limitations worth flagging to the LLM (the Agentfile's BIN block
// should restate these):
//   - only the `-c "script"` invocation is supported here; we
//     don't implement REPL or read-from-file modes.
//   - the mvdan interpreter doesn't support every obscure POSIX
//     feature (job control, terminal operations) — you won't miss
//     anything relevant to tool-chaining.
//   - the interpreter inherits the host environment and can reach
//     any path the process can. No sandboxing. Treat it as
//     equivalent to giving the LLM shell access on the host.
package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	script, err := argsToScript(args)
	if err != nil {
		return err
	}

	file, err := syntax.NewParser().Parse(strings.NewReader(script), "<stdin>")
	if err != nil {
		return fmt.Errorf("parse: %w", err)
	}

	runner, err := interp.New(
		interp.StdIO(os.Stdin, os.Stdout, os.Stderr),
	)
	if err != nil {
		return fmt.Errorf("interp: %w", err)
	}

	return runner.Run(context.Background(), file)
}

// argsToScript turns our argv into the script string the interpreter
// should run. Supports two call shapes:
//
//	sh -c "echo hello | tee out.txt"
//	sh "echo hello | tee out.txt"   (if the user omits -c)
//
// Anything else is an error — we deliberately reject the bare-repl
// case (`sh`) so a hallucinated tool call doesn't block waiting on
// stdin.
func argsToScript(args []string) (string, error) {
	switch len(args) {
	case 0:
		return "", fmt.Errorf("usage: sh -c \"<script>\"")
	case 1:
		return args[0], nil
	}

	if args[0] == "-c" {
		return strings.Join(args[1:], " "), nil
	}

	// Anything else is ambiguous; surface instead of guessing.
	return "", fmt.Errorf("usage: sh -c \"<script>\"")
}
