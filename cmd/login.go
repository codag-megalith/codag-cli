package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/codag-megalith/codag-cli/internal/config"
	"github.com/codag-megalith/codag-cli/internal/ui"
	"github.com/spf13/cobra"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Save your GitHub token",
	RunE: func(cmd *cobra.Command, args []string) error {
		scanner := bufio.NewScanner(os.Stdin)

		existing := config.GetToken()
		if existing != "" {
			fmt.Print("GITHUB_TOKEN already set. Replace? [y/N] ")
			if !scanner.Scan() {
				fmt.Println("\n\nSee ya!")
				return nil
			}
			answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
			if answer != "y" {
				ui.Info("Kept existing token.")
				return nil
			}
			fmt.Println()
		}

		fmt.Println("Create a token at: https://github.com/settings/tokens")
		fmt.Println("Required scope: repo (or public_repo for public repos)")
		fmt.Println()
		fmt.Print("Enter GITHUB_TOKEN: ")

		if !scanner.Scan() {
			fmt.Println("\n\nSee ya!")
			return nil
		}
		token := strings.TrimSpace(scanner.Text())
		if token == "" {
			ui.Error("No token provided.")
			return fmt.Errorf("no token provided")
		}

		if err := config.SaveEnvVar("GITHUB_TOKEN", token); err != nil {
			ui.Error(fmt.Sprintf("Could not save token: %s", err))
			return err
		}

		ui.Success(fmt.Sprintf("Saved to %s", config.EnvFile))
		return nil
	},
}
