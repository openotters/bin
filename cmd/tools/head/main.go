package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/openotters/bin/internal/wrap"
)

func main() {
	wrap.Run(func(args string) (string, error) {
		n := 10
		filename := args

		parts := strings.Fields(args)
		if len(parts) >= 3 && parts[0] == "-n" {
			var err error
			n, err = strconv.Atoi(parts[1])
			if err != nil {
				return "", fmt.Errorf("invalid line count: %w", err)
			}
			filename = parts[2]
		} else if len(parts) == 1 {
			filename = parts[0]
		}

		if filename == "" {
			return "", fmt.Errorf("usage: head [-n N] file")
		}

		f, err := os.Open(filename)
		if err != nil {
			return "", err
		}
		defer f.Close()

		var b strings.Builder
		scanner := bufio.NewScanner(f)

		for i := 0; i < n && scanner.Scan(); i++ {
			b.WriteString(scanner.Text())
			b.WriteByte('\n')
		}

		if err = scanner.Err(); err != nil {
			return "", err
		}

		return b.String(), nil
	})
}
