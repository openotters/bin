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
	wrap.Run(tail)
}

func tail(args string) (string, error) {
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
		return "", fmt.Errorf("usage: tail [-n N] file")
	}

	lines, err := readLines(filename)
	if err != nil {
		return "", err
	}

	start := len(lines) - n
	if start < 0 {
		start = 0
	}

	var b strings.Builder
	for _, line := range lines[start:] {
		b.WriteString(line)
		b.WriteByte('\n')
	}

	return b.String(), nil
}

func readLines(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	return lines, scanner.Err()
}
