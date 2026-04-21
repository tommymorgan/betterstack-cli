package main

import (
	"fmt"
	"os"

	"github.com/tommymorgan/betterstack-cli/cmd/betterstack"
	"github.com/tommymorgan/betterstack-cli/internal/errs"
)

func main() {
	err := betterstack.Execute()
	if err == nil {
		return
	}
	if !errs.IsSilent(err) {
		fmt.Fprintln(os.Stderr, "Error:", err)
	}
	os.Exit(int(errs.CodeOf(err)))
}
