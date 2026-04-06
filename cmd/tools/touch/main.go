package main

import (
	"github.com/openotters/bin/internal/wrap"
	"github.com/u-root/u-root/pkg/core/touch"
)

func main() {
	wrap.RunCommand(touch.New())
}
