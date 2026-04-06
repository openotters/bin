package main

import (
	"github.com/openotters/bin/internal/wrap"
	"github.com/u-root/u-root/pkg/core/cat"
)

func main() {
	wrap.RunCommand(cat.New())
}
