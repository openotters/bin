package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/openotters/bin/internal/cli"
)

func main() {
	cli.Run(func(args string) (string, error) {
		parts := strings.Fields(args)

		var start, end int
		var err error

		switch len(parts) {
		case 1:
			start = 1
			end, err = strconv.Atoi(parts[0])
			if err != nil {
				return "", fmt.Errorf("invalid number: %w", err)
			}
		case 2:
			start, err = strconv.Atoi(parts[0])
			if err != nil {
				return "", fmt.Errorf("invalid start: %w", err)
			}
			end, err = strconv.Atoi(parts[1])
			if err != nil {
				return "", fmt.Errorf("invalid end: %w", err)
			}
		default:
			return "", fmt.Errorf("usage: seq [start] end")
		}

		var b strings.Builder
		for i := start; i <= end; i++ {
			fmt.Fprintln(&b, i)
		}

		return b.String(), nil
	})
}
