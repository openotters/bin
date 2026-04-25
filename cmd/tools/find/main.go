package main

import (
	"github.com/openotters/bin/internal/cli"
	"github.com/u-root/u-root/pkg/core/find"
)

func main() {
	cli.Exec(find.New())
}
