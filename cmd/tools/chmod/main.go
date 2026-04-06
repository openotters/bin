package main

import (
	"github.com/openotters/bin/internal/wrap"
	"github.com/u-root/u-root/pkg/core/chmod"
)

func main() {
	wrap.RunCommand(chmod.New())
}
