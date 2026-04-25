package main

import (
	"github.com/openotters/bin/internal/cli"
	"github.com/u-root/u-root/pkg/core/chmod"
)

func main() {
	cli.Exec(chmod.New())
}
