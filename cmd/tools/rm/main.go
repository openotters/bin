package main

import (
	"github.com/openotters/bin/internal/wrap"
	"github.com/u-root/u-root/pkg/core/rm"
)

func main() {
	wrap.RunCommand(rm.New())
}
