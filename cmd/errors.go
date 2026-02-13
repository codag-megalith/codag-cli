package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/codag-megalith/codag-cli/internal/api"
	"github.com/codag-megalith/codag-cli/internal/ui"
)

// handleAPIError formats API errors for display and returns the error.
func handleAPIError(err error, server string) error {
	var apiErr *api.APIError
	if errors.As(err, &apiErr) {
		switch apiErr.StatusCode {
		case 401:
			ui.Error("Invalid or expired token. Run: codag login")
		case 404:
			ui.Error(fmt.Sprintf("Error %d: %s", apiErr.StatusCode, apiErr.Detail))
		default:
			ui.Error(fmt.Sprintf("Error %d: %s", apiErr.StatusCode, apiErr.Detail))
		}
	} else {
		ui.Error(fmt.Sprintf("Cannot connect to %s", server))
		fmt.Fprintln(os.Stderr, "  Check your connection or try again later.")
	}
	return err
}
