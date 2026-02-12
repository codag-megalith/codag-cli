package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	Version   = "dev"
	Commit    = "none"
	BuildDate = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		commit := Commit
		if len(commit) > 7 {
			commit = commit[:7]
		}
		fmt.Printf("codag %s (%s) built %s\n", Version, commit, BuildDate)
	},
}
