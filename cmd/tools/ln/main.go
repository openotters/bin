package main //nolint:cyclop // flag parsing adds complexity

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/openotters/bin/internal/wrap"
)

func main() {
	wrap.Run(func(args string) (string, error) {
		fields := strings.Fields(args)

		symlink := false
		force := false
		var positional []string

		for _, f := range fields {
			switch f {
			case "-s":
				symlink = true
			case "-f":
				force = true
			case "-sf", "-fs":
				symlink = true
				force = true
			default:
				positional = append(positional, f)
			}
		}

		if len(positional) < 1 || len(positional) > 2 {
			return "", fmt.Errorf("usage: ln [-sf] TARGET [LINK]")
		}

		target := positional[0]
		linkName := filepath.Base(target)
		if len(positional) == 2 {
			linkName = positional[1]
		}

		if force {
			os.Remove(linkName)
		}

		if symlink {
			return "", os.Symlink(target, linkName)
		}
		return "", os.Link(target, linkName)
	})
}
