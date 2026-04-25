package main

import (
	"path/filepath"
	"strings"

	"github.com/openotters/bin/internal/cli"
)

func main() {
	cli.Run(func(args string) (string, error) {
		fields := strings.Fields(args)
		name := filepath.Base(fields[0])
		if len(fields) > 1 && name != fields[1] {
			name = strings.TrimSuffix(name, fields[1])
		}
		return name, nil
	})
}
