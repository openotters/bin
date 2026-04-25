package main

import (
	"fmt"
	"time"

	"github.com/openotters/bin/internal/cli"
)

func main() {
	cli.Run(func(args string) (string, error) {
		d, err := time.ParseDuration(args)
		if err != nil {
			d, err = time.ParseDuration(args + "s")
		}
		if err != nil || d < 0 {
			return "", fmt.Errorf("invalid duration: %s", args)
		}
		time.Sleep(d)
		return fmt.Sprintf("slept %s", d), nil
	})
}
