package cmd

import (
	"fmt"
	"os"

	"github.com/codag-org/codag-cli/internal/api"
	"github.com/codag-org/codag-cli/internal/config"
	"github.com/codag-org/codag-cli/internal/ui"
	"github.com/spf13/cobra"
)

var indexCmd = &cobra.Command{
	Use:   "index",
	Short: "Re-index a registered repo",
	RunE: func(cmd *cobra.Command, args []string) error {
		token, err := config.RequireToken()
		if err != nil {
			ui.Error("GITHUB_TOKEN is required.")
			fmt.Fprintln(os.Stderr, "  Run: codag login")
			return err
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
				return fmt.Errorf("no repos")
			}
			last := repos[len(repos)-1]
			repoID = last.ID
			ui.Info(fmt.Sprintf("Using repo #%d (%s/%s)", last.ID, last.Owner, last.Name))
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
	addServerFlag(indexCmd)
}
