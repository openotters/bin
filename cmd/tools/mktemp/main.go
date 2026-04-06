package main

import (
	"github.com/openotters/bin/internal/wrap"
	"github.com/u-root/u-root/pkg/core/mktemp"
)

func main() {
	wrap.RunCommand(mktemp.New())
}
