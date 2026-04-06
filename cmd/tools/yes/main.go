package main

import (
	"strings"

	"github.com/openotters/bin/internal/wrap"
)

func main() {
	wrap.Run(func(args string) (string, error) {
		line := "y"
		if args != "" {
			line = args
		}

		var b strings.Builder
		for range 100 {
			b.WriteString(line)
			b.WriteByte('\n')
		}

		return b.String(), nil
	})
}
