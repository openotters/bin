package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/openotters/bin/internal/wrap"
)

func main() {
	wrap.Run(func(args string) (string, error) {
		var text string

		data, err := os.ReadFile(args)
		if err != nil {
			text = args
		} else {
			text = string(data)
		}

		lines := strings.Count(text, "\n")
		words := len(strings.Fields(text))
		chars := len(text)

		return fmt.Sprintf("%d lines %d words %d chars", lines, words, chars), nil
	})
}
