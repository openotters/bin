package main

import (
	"github.com/openotters/bin/internal/wrap"
	"github.com/u-root/u-root/pkg/core/xargs"
)

func main() {
	wrap.RunCommand(xargs.New())
}
