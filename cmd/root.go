package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/codag-megalith/codag-cli/internal/api"
	"github.com/codag-megalith/codag-cli/internal/config"
	"github.com/spf13/cobra"
)

var updateCheckDone = make(chan struct{})

var rootCmd = &cobra.Command{
	Use:   "codag",
	Short: "Organizational memory for coding agents",
	Long:  "Index your repo's PR history into safety signals for AI coding agents.",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		config.LoadEnv()

		// Background update check (non-blocking)
		if cmd.Name() != "upgrade" {
			startUpdateCheck(updateCheckDone)
		}

		// SIGINT / SIGTERM handler
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		go func() {
			<-sigCh
			fmt.Println("\n\nSee ya!")
			os.Exit(0)
		}()
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		if cmd.Name() != "upgrade" {
			<-updateCheckDone
			printUpdateNotice()
		}
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.AddCommand(loginCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(indexCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(upgradeCmd)
	rootCmd.AddCommand(mcpCmd)
}

// addServerFlag adds the hidden --server flag to a command.
func addServerFlag(cmd *cobra.Command) {
	cmd.Flags().String("server", "", "Override API server URL")
	cmd.Flags().MarkHidden("server")
}

// resolveServer returns the API base URL from flag > env > default.
func resolveServer(cmd *cobra.Command) string {
	if s, _ := cmd.Flags().GetString("server"); s != "" {
		return s
	}
	if s := config.GetServerURL(); s != "" {
		return s
	}
	return api.DefaultServer
}
