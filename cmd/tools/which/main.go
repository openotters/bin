package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/openotters/bin/internal/cli"
)

func main() {
	cli.Run(func(args string) (string, error) {
		fields := strings.Fields(args)
		allPaths := false
		var cmds []string

		for _, f := range fields {
			if f == "-a" {
				allPaths = true
			} else {
				cmds = append(cmds, f)
			}
		}

		if len(cmds) == 0 {
			return "", fmt.Errorf("expected command name")
		}

		paths := filepath.SplitList(os.Getenv("PATH"))
		var out []string

		for _, name := range cmds {
			for _, p := range paths {
				f := filepath.Join(p, name)
				info, err := os.Stat(f) //nolint:gosec // intended PATH lookup
				if err != nil {
					continue
				}
				if info.Mode()&0o111 != 0 {
					out = append(out, f)
					if !allPaths {
						break
					}
				}
			}
		}

		if len(out) == 0 {
			return "", fmt.Errorf("no suitable executable found")
		}
		return strings.Join(out, "\n"), nil
	})
}
