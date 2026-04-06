package main

import (
	"github.com/openotters/bin/internal/wrap"
	"github.com/u-root/u-root/pkg/core/cp"
)

func main() {
	wrap.RunCommand(cp.New())
}
