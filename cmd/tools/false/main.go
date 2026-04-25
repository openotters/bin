package main

import (
	"fmt"

	"github.com/openotters/bin/internal/cli"
)

func main() {
	cli.Run(func(_ string) (string, error) {
		return "", fmt.Errorf("false")
	})
}
