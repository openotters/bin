package main

import (
	"github.com/openotters/bin/internal/cli"
	"github.com/u-root/u-root/pkg/core/mktemp"
)

func main() {
	cli.Exec(mktemp.New())
}
