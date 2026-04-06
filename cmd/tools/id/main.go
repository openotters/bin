package main

import (
	"fmt"
	"os"
	"os/user"

	"github.com/openotters/bin/internal/wrap"
)

func main() {
	wrap.Run(func(_ string) (string, error) {
		u, err := user.Current()
		if err != nil {
			return "", err
		}

		return fmt.Sprintf("uid=%d(%s) gid=%d", os.Getuid(), u.Username, os.Getgid()), nil
	})
}
