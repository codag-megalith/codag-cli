package cmd

import (
	"fmt"
	"os"

	"github.com/codag-megalith/codag-cli/internal/api"
	"github.com/codag-megalith/codag-cli/internal/config"
	"github.com/codag-megalith/codag-cli/internal/ui"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show indexing status",
	RunE: func(cmd *cobra.Command, args []string) error {
		token, err := config.RequireToken()
		if err != nil {
			ui.Error("GITHUB_TOKEN is required.")
			fmt.Fprintln(os.Stderr, "  Run: codag login")
			return err
		}

		server := resolveServer(cmd)
		client := api.NewClient(server, token)

		repos, err := client.ListRepos()
		if err != nil {
			return handleAPIError(err, server)
		}

		if len(repos) == 0 {
			ui.Info("No repos registered. Run: codag init")
			return nil
		}

		fmt.Println()
		for _, repo := range repos {
			name := fmt.Sprintf("%s/%s", repo.Owner, repo.Name)
			indexed := "never"
			if repo.LastIndexedAt != nil {
				indexed = *repo.LastIndexedAt
				if len(indexed) > 10 {
					indexed = indexed[:10]
				}
			}

			stats, err := client.GetStats(repo.ID)
			if err != nil {
				// Silent error — print what we have
				fmt.Printf("  %s\n", ui.Bold.Render(name))
				ui.Keyval("Last indexed", indexed)
				fmt.Println()
				continue
			}

			fmt.Printf("  %s\n", ui.Bold.Render(name))
			ui.Keyval("Last indexed", indexed)
			fmt.Printf("  PRs: %d | Files w/ signals: %d | Signals: %d (%d danger)\n",
				stats.PRsIndexed, stats.FilesWithSignals, stats.TotalSignals, stats.DangerSignals)
			if stats.Indexing {
				fmt.Printf("  %s\n", ui.Yellow.Render("Status: indexing..."))
			}
			fmt.Println()
		}

		return nil
	},
}

func init() {
	addServerFlag(statusCmd)
}
