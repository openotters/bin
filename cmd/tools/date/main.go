package main

import (
	"strings"
	"time"

	"github.com/openotters/bin/internal/wrap"
)

//nolint:gochecknoglobals // POSIX format map
var fmtMap = map[string]string{
	"%a": "Mon", "%A": "Monday", "%b": "Jan", "%B": "January",
	"%d": "02", "%e": "_2", "%H": "15", "%I": "03", "%m": "1",
	"%M": "04", "%p": "PM", "%S": "05", "%y": "06", "%Y": "2006",
	"%z": "-0700", "%Z": "MST", "%c": time.UnixDate,
}

func main() {
	wrap.Run(func(args string) (string, error) {
		now := time.Now().UTC()
		if args == "" {
			return now.Format(time.RFC3339), nil
		}
		if strings.HasPrefix(args, "+") {
			format := args[1:]
			for k, v := range fmtMap {
				format = strings.ReplaceAll(format, k, v)
			}
			return now.Format(format), nil
		}
		return now.Format(time.RFC3339), nil
	})
}
