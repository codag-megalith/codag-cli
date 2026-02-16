package cmd

import (
	"fmt"
	"os"

	"github.com/codag-megalith/codag-cli/internal/api"
	"github.com/codag-megalith/codag-cli/internal/config"
	"github.com/codag-megalith/codag-cli/internal/ui"
	"github.com/spf13/cobra"
)

var indexCmd = &cobra.Command{
	Use:   "index",
	Short: "Re-index a registered repo",
	RunE: func(cmd *cobra.Command, args []string) error {
		token, err := config.RequireAuth()
		if err != nil {
			ui.Error("Not logged in.")
			fmt.Fprintln(os.Stderr, "  Run: codag login")
			return silent(err)
		}

		server := resolveServer(cmd)
		client := api.NewClient(server, token)

		repoID, _ := cmd.Flags().GetInt("repo")
		if repoID == 0 {
			repos, err := client.ListRepos()
			if err != nil {
				return handleAPIError(err, server)
			}
			if len(repos) == 0 {
				ui.Error("No repos registered. Run: codag init")
				return silent(fmt.Errorf("no repos"))
			}
			last := repos[len(repos)-1]
			repoID = last.ID
			ui.Info(fmt.Sprintf("Using repo #%d (%s/%s)", last.ID, last.Owner, last.Name))
		}

		// Re-indexing is expensive — require explicit confirmation
		force, _ := cmd.Flags().GetBool("force")
		if !force {
			fmt.Println()
			ui.Warn("Re-indexing deletes all existing data and can take up to hours.")
			fmt.Println("  This re-processes all PRs from scratch. You usually don't need this —")
			fmt.Println("  new PRs are indexed automatically via webhooks.")
			fmt.Println()
			fmt.Print("  Continue? [y/N] ")
			var answer string
			fmt.Scanln(&answer)
			if answer != "y" && answer != "Y" {
				ui.Info("Cancelled.")
				return nil
			}
			fmt.Println()
		}

		ui.Info("Indexing PR history...")

		maxPRs, _ := cmd.Flags().GetInt("max-prs")
		var maxPRsPtr *int
		if maxPRs > 0 {
			maxPRsPtr = &maxPRs
		}

		result, err := client.TriggerBackfill(repoID, maxPRsPtr)
		if err != nil {
			return handleAPIError(err, server)
		}

		if result.Status == "already_running" {
			ui.Warn("Indexing already in progress.")
		}

		_, err = pollIndexing(client, repoID)
		return err
	},
}

func init() {
	indexCmd.Flags().Int("repo", 0, "Repo ID (default: most recent)")
	indexCmd.Flags().Int("max-prs", 0, "Max PRs to fetch")
	indexCmd.Flags().BoolP("force", "f", false, "Skip confirmation prompt")
	addServerFlag(indexCmd)
}
