package cmd

import (
	"fmt"
	"strings"

	"github.com/codag-megalith/codag-cli/internal/api"
	"github.com/codag-megalith/codag-cli/internal/config"
	"github.com/codag-megalith/codag-cli/internal/ui"
	"github.com/spf13/cobra"
)

var accountCmd = &cobra.Command{
	Use:   "account",
	Short: "Show account info and current plan",
	RunE: func(cmd *cobra.Command, args []string) error {
		if !config.HasAuth() {
			ui.Error("Not logged in. Run `codag login` first.")
			return silent(fmt.Errorf("not authenticated"))
		}

		server := resolveServer(cmd)
		client := api.NewClient(server, config.GetAccessToken())

		me, err := client.GetMe()
		if err != nil {
			if apiErr, ok := err.(*api.APIError); ok && apiErr.StatusCode == 401 {
				ui.Error("Session expired. Run `codag login` to re-authenticate.")
				return silent(err)
			}
			return err
		}

		fmt.Println()
		ui.Keyval("User", me.User.GithubLogin)
		if me.User.Email != "" {
			ui.Keyval("Email", me.User.Email)
		}

		// Plan
		tier := "Free"
		if me.Subscription != nil && me.Subscription.Tier != "" {
			tier = strings.ToUpper(me.Subscription.Tier[:1]) + me.Subscription.Tier[1:]
		}
		ui.Keyval("Plan", tier)

		if me.Subscription != nil && me.Subscription.CancelAtPeriodEnd {
			ui.Warn("Cancellation pending — reverts to Free at period end")
		}

		// Repos
		ui.Keyval("Repos", fmt.Sprintf("%d", len(me.Repos)))

		// Orgs
		if len(me.Orgs) > 0 {
			names := make([]string, len(me.Orgs))
			for i, o := range me.Orgs {
				names[i] = o.Name
			}
			ui.Keyval("Orgs", strings.Join(names, ", "))
		}

		fmt.Println()
		return nil
	},
}

func init() {
	addServerFlag(accountCmd)
}
