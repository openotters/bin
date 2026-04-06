package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/openotters/bin/internal/wrap"
)

func main() {
	wrap.Run(func(args string) (string, error) {
		fields := strings.Fields(args)
		if len(fields) == 0 {
			return "", fmt.Errorf("missing operand")
		}
		var out []string
		for _, arg := range fields {
			abs, err := filepath.Abs(arg)
			if err != nil {
				return "", err
			}
			resolved, err := filepath.EvalSymlinks(abs)
			if err != nil {
				return "", err
			}
			out = append(out, filepath.Clean(resolved))
		}
		return strings.Join(out, "\n"), nil
	})
}
