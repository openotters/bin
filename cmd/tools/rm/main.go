package main

import (
	"github.com/openotters/bin/internal/cli"
	"github.com/u-root/u-root/pkg/core/rm"
)

func main() {
	cli.Exec(rm.New())
}
