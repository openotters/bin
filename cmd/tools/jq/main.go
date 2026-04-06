package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/itchyny/gojq"

	"github.com/openotters/bin/internal/wrap"
)

func main() {
	wrap.Run(func(args string) (string, error) {
		parts := strings.SplitN(args, "\n", 2)
		if len(parts) < 2 {
			return "", fmt.Errorf("usage: first line is the jq expression, rest is JSON input")
		}

		expr := strings.TrimSpace(parts[0])
		input := strings.TrimSpace(parts[1])

		query, err := gojq.Parse(expr)
		if err != nil {
			return "", fmt.Errorf("parse error: %w", err)
		}

		var data any
		if err = json.Unmarshal([]byte(input), &data); err != nil {
			return "", fmt.Errorf("invalid JSON: %w", err)
		}

		var b strings.Builder
		iter := query.Run(data)

		for v, ok := iter.Next(); ok; v, ok = iter.Next() {
			if jqErr, isErr := v.(error); isErr {
				return "", jqErr
			}

			out, marshalErr := json.MarshalIndent(v, "", "  ")
			if marshalErr != nil {
				return "", marshalErr
			}

			b.Write(out)
			b.WriteByte('\n')
		}

		return strings.TrimSpace(b.String()), nil
	})
}
