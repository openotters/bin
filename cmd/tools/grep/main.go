package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/openotters/bin/internal/cli"
)

func main() {
	cli.Run(func(args string) (string, error) {
		parts := strings.SplitN(args, " ", 2)
		if len(parts) < 2 {
			return "", fmt.Errorf("usage: grep pattern file")
		}

		pattern, filename := parts[0], parts[1]

		re, err := regexp.Compile(pattern)
		if err != nil {
			return "", fmt.Errorf("invalid pattern: %w", err)
		}

		f, err := os.Open(filename)
		if err != nil {
			return "", err
		}
		defer f.Close()

		var b strings.Builder
		scanner := bufio.NewScanner(f)

		for scanner.Scan() {
			if re.MatchString(scanner.Text()) {
				b.WriteString(scanner.Text())
				b.WriteByte('\n')
			}
		}

		if err = scanner.Err(); err != nil {
			return "", err
		}

		return b.String(), nil
	})
}
