package main

import (
	"github.com/openotters/bin/internal/wrap"
	"github.com/u-root/u-root/pkg/core/shasum"
)

func main() {
	wrap.RunCommand(shasum.New())
}
