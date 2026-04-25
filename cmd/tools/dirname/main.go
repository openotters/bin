package main

import (
	"path/filepath"
	"strings"

	"github.com/openotters/bin/internal/cli"
)

func main() {
	cli.Run(func(args string) (string, error) {
		var out []string
		for _, arg := range strings.Fields(args) {
			out = append(out, filepath.Dir(arg))
		}
		return strings.Join(out, "\n"), nil
	})
}
