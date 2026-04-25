package main

import (
	"github.com/openotters/bin/internal/cli"
	"github.com/u-root/u-root/pkg/core/shasum"
)

func main() {
	cli.Exec(shasum.New())
}
