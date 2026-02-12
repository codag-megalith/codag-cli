package main

import (
	"os"

	"github.com/codag-org/codag-cli/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
