package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/openotters/bin/internal/wrap"
)

func main() {
	wrap.Run(func(args string) (string, error) {
		dirs := strings.Fields(args)
		if len(dirs) == 0 {
			return "", fmt.Errorf("rmdir requires a directory argument")
		}

		for _, dir := range dirs {
			if err := os.Remove(dir); err != nil {
				return "", err
			}
		}

		return "", nil
	})
}
