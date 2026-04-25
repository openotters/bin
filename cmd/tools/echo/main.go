package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	fmt.Fprintln(os.Stdout, strings.Join(os.Args[1:], " "))
	os.Exit(0)
}
