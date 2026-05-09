// yaegi runs Go source under the embedded yaegi interpreter and prints
// whatever the program writes to stdout. Designed to be invoked through
// the openotters BIN runtime: source comes from argv (joined into a
// single string with single spaces) or, when no argv is given, from
// stdin. Snippets that don't declare `package main` are auto-wrapped
// so the agent can write one-liners like `fmt.Println("hi")` directly,
// with a curated set of stdlib imports already in scope.
//
// We can't ship upstream yaegi as a vendored binary because its CLI
// only accepts source via a file path or `run -` stdin — neither
// matches the BIN runtime's argv-in / stdout-out contract.
package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"
)

const wrapper = `package main

import (
	"fmt"
	"os"
	"strings"
	"strconv"
	"sort"
	"math"
	"encoding/json"
	"regexp"
)

// Suppress "imported and not used" — agents won't always touch every import.
var _ = fmt.Sprintln
var _ = os.Args
var _ = strings.Builder{}
var _ = strconv.Itoa
var _ = sort.Strings
var _ = math.Pi
var _ = json.Marshal
var _ = regexp.MustCompile

func main() {
%s
}
`

func main() {
	src, err := readSource()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	src = strings.TrimSpace(src)
	if src == "" {
		fmt.Fprintln(os.Stderr, "yaegi: no Go source supplied (pass via argv or stdin)")
		os.Exit(1)
	}

	if !strings.HasPrefix(src, "package ") {
		src = fmt.Sprintf(wrapper, src)
	}

	i := interp.New(interp.Options{
		Args:   os.Args,
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	})

	if err := i.Use(stdlib.Symbols); err != nil {
		fmt.Fprintln(os.Stderr, "yaegi: load stdlib:", err)
		os.Exit(1)
	}

	// yaegi's Eval of a `package main` source defines AND invokes main()
	// in one step. Calling main() explicitly afterwards would double-run.
	if _, err := i.Eval(src); err != nil {
		fmt.Fprintln(os.Stderr, "yaegi:", err)
		os.Exit(1)
	}
}

func readSource() (string, error) {
	if len(os.Args) > 1 {
		return strings.Join(os.Args[1:], " "), nil
	}

	b, err := io.ReadAll(os.Stdin)
	if err != nil {
		return "", fmt.Errorf("read stdin: %w", err)
	}

	return string(b), nil
}
