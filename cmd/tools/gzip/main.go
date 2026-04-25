package main

import (
	"github.com/openotters/bin/internal/cli"
	"github.com/u-root/u-root/pkg/core/gzip"
)

func main() {
	cli.Exec(gzip.New("gzip"))
}
