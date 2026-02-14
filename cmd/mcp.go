package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	codagmcp "github.com/codag-megalith/codag-cli/internal/mcp"
	"github.com/codag-megalith/codag-cli/internal/ui"
	"github.com/spf13/cobra"
	"golang.org/x/term"
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
		// Detect interactive terminal — MCP servers are meant to be launched by an IDE, not run directly
		if term.IsTerminal(int(os.Stdin.Fd())) {
			ui.Warn("This command starts an MCP server over stdio (JSON-RPC).")
			fmt.Println("  It's meant to be launched by your IDE (Cursor, VS Code, etc.), not run directly.")
			fmt.Println()
			fmt.Println("  To set up MCP for a project, run:")
			fmt.Printf("    %s\n", ui.Bold.Render("codag init"))
			fmt.Println()
			fmt.Println("  This adds the MCP config to your project so your editor picks it up automatically.")
			return nil
		}

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
