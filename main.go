package main

import (
	"fmt"
	"os"

	"github.com/tommymorgan/betterstack-cli/cmd/betterstack"
)

func main() {
	if err := betterstack.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
