package main

import (
	"encoding/json"
	"fmt"
	"os"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	var input struct {
		Input string `json:"input"`
	}

	if err := json.NewDecoder(os.Stdin).Decode(&input); err != nil {
		return fmt.Errorf("decoding input: %w", err)
	}

	return json.NewEncoder(os.Stdout).Encode(map[string]string{
		"output": input.Input,
	})
}
