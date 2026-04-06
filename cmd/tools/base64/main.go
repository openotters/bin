package main

import (
	"github.com/openotters/bin/internal/wrap"
	"github.com/u-root/u-root/pkg/core/base64"
)

func main() {
	wrap.RunCommand(base64.New())
}
