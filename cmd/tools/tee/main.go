package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/openotters/bin/internal/cli"
)

func main() {
	cli.Run(func(args string) (string, error) {
		lines := strings.SplitN(args, "\n", 2)
		if len(lines) < 2 {
			return "", fmt.Errorf("usage: first line is filename(s), rest is content")
		}

		header := strings.Fields(lines[0])
		content := lines[1]
		appendMode := false
		var files []string

		for _, h := range header {
			if h == "-a" {
				appendMode = true
			} else {
				files = append(files, h)
			}
		}

		flag := os.O_WRONLY | os.O_CREATE
		if appendMode {
			flag |= os.O_APPEND
		} else {
			flag |= os.O_TRUNC
		}

		for _, fname := range files {
			f, err := os.OpenFile(fname, flag, 0o644)
			if err != nil {
				return "", err
			}
			_, _ = f.WriteString(content)
			f.Close()
		}

		return content, nil
	})
}
