package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/codag-megalith/codag-cli/internal/api"
	"github.com/codag-megalith/codag-cli/internal/config"
	"github.com/codag-megalith/codag-cli/internal/mcpconfig"
	"github.com/codag-megalith/codag-cli/internal/ui"
	"github.com/spf13/cobra"
)

var sshRemoteRe = regexp.MustCompile(`^git@github\.com:(.+?)(?:\.git)?$`)

var initCmd = &cobra.Command{
	Use:   "init [github-url]",
	Short: "Register a repo and start indexing",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		token, err := config.RequireAuth()
		if err != nil {
			ui.Error("Not logged in.")
			fmt.Fprintln(os.Stderr, "  Run: codag login")
			return silent(err)
		}

		server := resolveServer(cmd)
		client := api.NewClient(server, token)
		scanner := bufio.NewScanner(os.Stdin)

		// Resolve GitHub URL
		var githubURL string
		var repoRoot string

		if len(args) > 0 {
			githubURL = args[0]
			// For explicit URLs, use cwd for .mcp.json
			repoRoot, _ = os.Getwd()
		} else {
			githubURL, repoRoot = detectGitHubURL()
			if githubURL == "" {
				ui.Error("Not in a git repo with a GitHub remote.")
				fmt.Fprintln(os.Stderr, "  Usage: codag init <github-url>")
				return silent(fmt.Errorf("no github URL detected"))
			}

			fmt.Printf("Detected: %s\n", githubURL)
			fmt.Print("Index this repo? [Y/n] ")
			if scanner.Scan() {
				answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
				if answer != "" && answer != "y" {
					ui.Info("Cancelled.")
					return nil
				}
			}
		}

		// Register repo
		fmt.Println()
		ui.Info(fmt.Sprintf("Registering %s...", githubURL))

		repo, err := client.RegisterRepo(githubURL)
		if err != nil {
			return handleAPIError(err, server)
		}

		// Setup webhook (non-blocking — failures warn but don't abort)
		setupWebhook(client, repo.ID)

		if repo.LastIndexedAt != nil {
			indexed := *repo.LastIndexedAt
			if len(indexed) > 10 {
				indexed = indexed[:10]
			}
			ui.Warn(fmt.Sprintf("Already indexed (last: %s)", indexed))
			fmt.Printf("\n  To re-index: codag index --repo %d\n", repo.ID)

			// Still write .mcp.json even if already indexed
			writeMCPConfig(repoRoot, server)
			return nil
		}

		ui.Success(fmt.Sprintf("Registered: %s/%s (id: %d)", repo.Owner, repo.Name, repo.ID))

		// Trigger backfill
		fmt.Println()
		ui.Info("Indexing PR history...")

		maxPRs, _ := cmd.Flags().GetInt("max-prs")
		var maxPRsPtr *int
		if maxPRs > 0 {
			maxPRsPtr = &maxPRs
		}

		_, err = client.TriggerBackfill(repo.ID, maxPRsPtr)
		if err != nil {
			return handleAPIError(err, server)
		}

		// Poll until done
		_, err = pollIndexing(client, repo.ID)
		if err != nil {
			return err
		}

		// Write .mcp.json
		fmt.Println()
		writeMCPConfig(repoRoot, server)

		return nil
	},
}

func init() {
	initCmd.Flags().Int("max-prs", 0, "Max PRs to fetch (default: 500)")
	addServerFlag(initCmd)
}

// detectGitHubURL runs git commands to find the GitHub remote URL and repo root.
func detectGitHubURL() (string, string) {
	// Get repo root
	rootCmd := exec.Command("git", "rev-parse", "--show-toplevel")
	rootOut, err := rootCmd.Output()
	if err != nil {
		return "", ""
	}
	repoRoot := strings.TrimSpace(string(rootOut))

	// Get remote URL
	remoteCmd := exec.Command("git", "remote", "get-url", "origin")
	out, err := remoteCmd.Output()
	if err != nil {
		return "", ""
	}
	remote := strings.TrimSpace(string(out))

	// Parse SSH format: git@github.com:owner/repo.git
	if m := sshRemoteRe.FindStringSubmatch(remote); m != nil {
		return "https://github.com/" + m[1], repoRoot
	}

	// Parse HTTPS format
	if strings.Contains(remote, "github.com") {
		url := strings.TrimSuffix(remote, ".git")
		return url, repoRoot
	}

	return "", ""
}

// setupWebhook attempts to create a GitHub webhook for auto-reindexing.
// Failures are non-fatal — we warn and continue.
func setupWebhook(client *api.Client, repoID int) {
	webhookResp, err := client.SetupWebhook(repoID)
	if err != nil {
		if apiErr, ok := err.(*api.APIError); ok {
			switch apiErr.StatusCode {
			case 400:
				ui.Warn("No GitHub token stored. Webhook skipped.")
				fmt.Fprintln(os.Stderr, "  Log in at console.codag.ai to enable auto-reindexing.")
			case 403:
				ui.Warn("No admin access to this repo. Webhook skipped.")
			default:
				ui.Warn("Webhook setup failed: " + apiErr.Detail)
			}
		} else {
			ui.Warn("Webhook setup failed: " + err.Error())
		}
		return
	}

	switch webhookResp.Status {
	case "created":
		ui.Success("Webhook created — auto-reindex on push, PRs, and issues")
	case "already_exists":
		ui.Info("Webhook already configured")
	}
}

// writeMCPConfig writes MCP configs for all detected editors.
func writeMCPConfig(repoRoot string, serverURL string) {
	if repoRoot == "" {
		return
	}

	results := mcpconfig.WriteAll(repoRoot, serverURL)

	if len(results) == 0 {
		ui.Warn("Could not write MCP config.")
		ui.Info("Add this to your .mcp.json manually:")
		ui.CodeBlock(fmt.Sprintf(`"codag": {
  "command": "codag",
  "args": ["mcp", "serve", "."],
  "env": { "CODAG_URL": "%s" }
}`, serverURL))
		return
	}

	for _, r := range results {
		switch r.Action {
		case "created":
			ui.Success(fmt.Sprintf("Created %s (%s)", r.Path, r.Editor))
		case "updated":
			ui.Success(fmt.Sprintf("Updated %s (%s)", r.Path, r.Editor))
		case "unchanged":
			ui.Info(fmt.Sprintf("%s already configured (%s)", r.Path, r.Editor))
		}
	}
	if strings.Contains(serverURL, "localhost") || strings.Contains(serverURL, "127.0.0.1") {
		ui.Warn("MCP config points to a local dev server.")
		fmt.Println("  Re-run 'codag init' without --dev before committing.")
	} else {
		fmt.Println("  Your coding agent now has access to Codag signals.")
	}
}
