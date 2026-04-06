package main

import (
	"fmt"
	"net"
	"time"

	"github.com/openotters/bin/internal/wrap"
)

func main() {
	wrap.Run(func(args string) (string, error) {
		if args == "" {
			return "", fmt.Errorf("usage: ping host")
		}

		dialer := net.Dialer{Timeout: 5 * time.Second}
		conn, err := dialer.Dial("tcp", net.JoinHostPort(args, "80"))
		if err != nil {
			return "", fmt.Errorf("%s is unreachable: %w", args, err)
		}
		conn.Close()

		return fmt.Sprintf("%s is reachable", args), nil
	})
}
