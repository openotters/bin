package main

import (
	"github.com/openotters/bin/internal/cli"
	"github.com/u-root/u-root/pkg/core/xargs"
)

func main() {
	cli.Exec(xargs.New())
}
