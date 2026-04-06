package main

import "github.com/openotters/bin/internal/wrap"

func main() {
	wrap.Run(func(_ string) (string, error) {
		return "", nil
	})
}
