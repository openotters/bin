package main

import (
	"github.com/openotters/bin/internal/cli"
	"github.com/u-root/u-root/pkg/core/touch"
)

func main() {
	cli.Exec(touch.New())
}
