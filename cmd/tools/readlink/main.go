package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/openotters/bin/internal/wrap"
)

func main() {
	wrap.Run(func(args string) (string, error) {
		fields := strings.Fields(args)
		follow := false
		var files []string

		for _, f := range fields {
			if f == "-f" {
				follow = true
			} else {
				files = append(files, f)
			}
		}

		if len(files) == 0 {
			return "", fmt.Errorf("missing operand")
		}

		var out []string
		for _, file := range files {
			if follow {
				p, err := filepath.EvalSymlinks(file)
				if err != nil {
					return "", err
				}
				out = append(out, p)
			} else {
				p, err := os.Readlink(file)
				if err != nil {
					return "", err
				}
				out = append(out, p)
			}
		}
		return strings.Join(out, "\n"), nil
	})
}
