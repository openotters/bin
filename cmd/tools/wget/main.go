package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/openotters/bin/internal/cli"
)

// requestTimeout bounds a single wget fetch. Agents invoke this tool
// inside their own step budget; a runaway URL must not stall the
// whole agent on DNS / slow-body / tarpit endpoints.
const requestTimeout = 30 * time.Second

func main() {
	cli.Run(func(args string) (string, error) {
		if args == "" {
			return "", fmt.Errorf("usage: wget url")
		}

		ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, args, nil)
		if err != nil {
			return "", fmt.Errorf("creating request: %w", err)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024))
		if err != nil {
			return "", err
		}

		return string(body), nil
	})
}
