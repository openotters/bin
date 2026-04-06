package main

import (
	"github.com/openotters/bin/internal/wrap"
	"github.com/u-root/u-root/pkg/core/ls"
)

func main() {
	wrap.RunCommand(ls.New())
}
