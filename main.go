package main

import (
	"fmt"
	"os"

	"github.com/codag-megalith/codag-cli/cmd"
	"github.com/codag-megalith/codag-cli/internal/ui"
)

func main() {
	if err := cmd.Execute(); err != nil {
		// SilenceErrors is true, so cobra won't print — we do it here.
		if msg := err.Error(); msg != "" {
			ui.Error(fmt.Sprintf("%s", msg))
		}
		os.Exit(1)
	}
}
