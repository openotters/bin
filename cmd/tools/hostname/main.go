package main

import (
	"os"

	"github.com/openotters/bin/internal/wrap"
)

func main() {
	wrap.Run(func(_ string) (string, error) {
		return os.Hostname()
	})
}
