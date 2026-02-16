package main

import (
	"fmt"
	"os"

	"github.com/codag-megalith/codag-cli/cmd"
	"github.com/codag-megalith/codag-cli/internal/ui"
)

func main() {
	if err := cmd.Execute(); err != nil {
		if !cmd.IsSilent(err) {
			ui.Error(fmt.Sprintf("%s", err))
		}
		os.Exit(1)
	}
}
