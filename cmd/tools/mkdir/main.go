package main

import (
	"github.com/openotters/bin/internal/wrap"
	"github.com/u-root/u-root/pkg/core/mkdir"
)

func main() {
	wrap.RunCommand(mkdir.New())
}
