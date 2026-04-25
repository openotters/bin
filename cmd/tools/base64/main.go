package main

import (
	"github.com/openotters/bin/internal/cli"
	"github.com/u-root/u-root/pkg/core/base64"
)

func main() {
	cli.Exec(base64.New())
}
