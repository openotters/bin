package main

import (
	"os"

	"github.com/openotters/bin/internal/cli"
)

func main() {
	cli.Run(func(args string) (string, error) {
		data, err := os.ReadFile(args)
		if err != nil {
			return "", err
		}

		return string(data), nil
	})
}
