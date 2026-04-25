// Package cli is the thin entry-point helper BIN tool mains use to
// behave like plain CLIs. It replaced the older wrap package (which
// implemented a JSON-over-stdin envelope) now that the runtime's
// tool executor dispatches argv directly — tools no longer need to
// know anything about the LLM tool-call protocol.
package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/u-root/u-root/pkg/core"
)

// Exec runs a u-root core.Command as a standalone program: stdin,
// stdout, and stderr pass through to the process; argv forwards from
// os.Args[1:]; a non-zero exit is emitted on any error the command
// returns. Kept as a one-liner so each BIN main stays a trivial
// wrapper.
func Exec(cmd core.Command) {
	cmd.SetIO(os.Stdin, os.Stdout, os.Stderr)

	if err := cmd.Run(os.Args[1:]...); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// Run is the plain-CLI entry point for BIN tools that previously used
// wrap.Run. fn receives os.Args[1:] joined with a single space — the
// same shape the old stdin-JSON envelope surfaced — so each tool's
// internal parsing stays unchanged. Errors from fn go to stderr and
// produce a non-zero exit; fn's string output is printed verbatim
// (no trailing newline added).
//
// Tools with structured arg needs (multi-positional, quoted strings,
// flags) should read os.Args directly instead of routing through
// Run. This helper is a convenience for the common free-form-string
// case only.
func Run(fn func(args string) (string, error)) {
	args := strings.Join(os.Args[1:], " ")

	out, err := fn(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Fprint(os.Stdout, out)
}
