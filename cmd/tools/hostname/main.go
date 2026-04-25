package main

import (
	"os"

	"github.com/openotters/bin/internal/cli"
)

func main() {
	cli.Run(func(_ string) (string, error) {
		return os.Hostname()
	})
}
