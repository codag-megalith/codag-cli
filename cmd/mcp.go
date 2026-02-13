package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	codagmcp "github.com/codag-megalith/codag-cli/internal/mcp"
	"github.com/spf13/cobra"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "MCP server commands",
}

var mcpServeCmd = &cobra.Command{
	Use:   "serve [workspace-path]",
	Short: "Start the Codag MCP server (stdio)",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		workspace := "."
		if len(args) > 0 {
			workspace = args[0]
		}

		absPath, err := filepath.Abs(workspace)
		if err != nil {
			return fmt.Errorf("invalid workspace path: %w", err)
		}

		if _, err := os.Stat(absPath); os.IsNotExist(err) {
			return fmt.Errorf("workspace path does not exist: %s", absPath)
		}

		server := resolveServer(cmd)
		return codagmcp.Serve(absPath, server, Version)
	},
}

func init() {
	addServerFlag(mcpServeCmd)
	mcpCmd.AddCommand(mcpServeCmd)
}
