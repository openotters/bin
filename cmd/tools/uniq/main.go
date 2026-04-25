package main //nolint:cyclop // flag parsing adds complexity

import (
	"fmt"
	"strings"

	"github.com/openotters/bin/internal/cli"
)

func main() {
	cli.Run(func(args string) (string, error) {
		lines := strings.Split(args, "\n")

		countMode := false
		dupsOnly := false
		uniqOnly := false
		ignoreCase := false
		var input []string

		for i, line := range lines {
			if i == 0 && strings.HasPrefix(line, "-") {
				for _, c := range line[1:] {
					switch c {
					case 'c':
						countMode = true
					case 'd':
						dupsOnly = true
					case 'u':
						uniqOnly = true
					case 'i':
						ignoreCase = true
					}
				}
				continue
			}
			input = append(input, line)
		}

		if len(input) == 0 {
			return "", nil
		}

		equal := func(a, b string) bool {
			if ignoreCase {
				return strings.EqualFold(a, b)
			}
			return a == b
		}

		type group struct {
			line  string
			count int
		}

		var groups []group
		groups = append(groups, group{input[0], 1})
		for _, line := range input[1:] {
			if equal(line, groups[len(groups)-1].line) {
				groups[len(groups)-1].count++
			} else {
				groups = append(groups, group{line, 1})
			}
		}

		var out []string
		for _, g := range groups {
			if dupsOnly && g.count < 2 {
				continue
			}
			if uniqOnly && g.count > 1 {
				continue
			}
			if countMode {
				out = append(out, fmt.Sprintf("%d\t%s", g.count, g.line))
			} else {
				out = append(out, g.line)
			}
		}
		return strings.Join(out, "\n"), nil
	})
}
