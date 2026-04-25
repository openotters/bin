package main

import (
	"fmt"
	"strings"

	"github.com/openotters/bin/internal/cli"
)

func main() {
	cli.Run(func(args string) (string, error) {
		lines := strings.SplitN(args, "\n", 3)

		deleteMode := false
		if len(lines) >= 1 && lines[0] == "-d" {
			deleteMode = true
			lines = lines[1:]
		}

		if deleteMode {
			if len(lines) < 2 {
				return "", fmt.Errorf("usage: -d\\nSET1\\nINPUT")
			}
			set := lines[0]
			input := lines[1]
			for _, c := range set {
				input = strings.ReplaceAll(input, string(c), "")
			}
			return input, nil
		}

		if len(lines) < 3 {
			return "", fmt.Errorf("usage: SET1\\nSET2\\nINPUT")
		}

		from, to, input := lines[0], lines[1], lines[2]
		var pairs []string
		for i := 0; i < len(from) && i < len(to); i++ {
			pairs = append(pairs, string(from[i]), string(to[i]))
		}
		return strings.NewReplacer(pairs...).Replace(input), nil
	})
}
