package main

import (
	"github.com/openotters/bin/internal/cli"
	"github.com/u-root/u-root/pkg/core/mv"
)

func main() {
	cli.Exec(mv.New())
}
