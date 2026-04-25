package main

import (
	"github.com/openotters/bin/internal/cli"
	"github.com/u-root/u-root/pkg/core/cp"
)

func main() {
	cli.Exec(cp.New())
}
