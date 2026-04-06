package main

import (
	"os"
	"strings"

	"github.com/openotters/bin/internal/wrap"
)

func main() {
	wrap.Run(func(args string) (string, error) {
		if args == "" {
			return strings.Join(os.Environ(), "\n"), nil
		}
		var out []string
		for _, name := range strings.Fields(args) {
			if v, ok := os.LookupEnv(name); ok {
				out = append(out, v)
			}
		}
		return strings.Join(out, "\n"), nil
	})
}
