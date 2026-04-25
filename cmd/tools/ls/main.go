package main

import (
	"github.com/openotters/bin/internal/cli"
	"github.com/u-root/u-root/pkg/core/ls"
)

func main() {
	cli.Exec(ls.New())
}
