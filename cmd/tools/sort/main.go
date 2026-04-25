package main

import (
	"os"
	"sort"
	"strings"

	"github.com/openotters/bin/internal/cli"
)

func main() {
	cli.Run(func(args string) (string, error) {
		var text string

		data, err := os.ReadFile(args)
		if err != nil {
			text = args
		} else {
			text = string(data)
		}

		lines := strings.Split(strings.TrimRight(text, "\n"), "\n")
		sort.Strings(lines)

		return strings.Join(lines, "\n") + "\n", nil
	})
}
