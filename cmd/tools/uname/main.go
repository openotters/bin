package main

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/openotters/bin/internal/wrap"
)

func main() {
	wrap.Run(func(args string) (string, error) {
		hostname, _ := os.Hostname()
		sysname := runtime.GOOS
		machine := runtime.GOARCH
		release := runtime.Version()

		if args == "" || args == "-s" {
			return sysname, nil
		}

		if args == "-a" {
			return fmt.Sprintf("%s %s %s %s", sysname, hostname, release, machine), nil
		}

		var parts []string
		for _, c := range args {
			switch c {
			case 's':
				parts = append(parts, sysname)
			case 'n':
				parts = append(parts, hostname)
			case 'r':
				parts = append(parts, release)
			case 'm', 'p':
				parts = append(parts, machine)
			}
		}
		if len(parts) == 0 {
			return sysname, nil
		}
		return strings.Join(parts, " "), nil
	})
}
