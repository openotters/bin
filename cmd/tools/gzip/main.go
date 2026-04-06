package main

import (
	"github.com/openotters/bin/internal/wrap"
	"github.com/u-root/u-root/pkg/core/gzip"
)

func main() {
	wrap.RunCommand(gzip.New("gzip"))
}
