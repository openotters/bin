package main

import "github.com/openotters/bin/internal/cli"

func main() {
	cli.Run(func(_ string) (string, error) {
		return "", nil
	})
}
