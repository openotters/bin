package main

import (
	"github.com/openotters/bin/internal/wrap"
	"github.com/u-root/u-root/pkg/core/mv"
)

func main() {
	wrap.RunCommand(mv.New())
}
