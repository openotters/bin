package main

import (
	"github.com/openotters/bin/internal/wrap"
	"github.com/u-root/u-root/pkg/core/find"
)

func main() {
	wrap.RunCommand(find.New())
}
