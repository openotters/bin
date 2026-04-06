package main

import (
	"time"

	"github.com/openotters/bin/internal/wrap"
)

func main() {
	wrap.Run(func(_ string) (string, error) {
		return time.Now().Format(time.RFC3339), nil
	})
}
