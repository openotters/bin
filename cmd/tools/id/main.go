package main

import (
	"fmt"
	"os"
	"os/user"

	"github.com/openotters/bin/internal/cli"
)

func main() {
	cli.Run(func(_ string) (string, error) {
		u, err := user.Current()
		if err != nil {
			return "", err
		}

		return fmt.Sprintf("uid=%d(%s) gid=%d", os.Getuid(), u.Username, os.Getgid()), nil
	})
}
