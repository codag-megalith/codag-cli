package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/codag-megalith/codag-cli/internal/api"
	"github.com/codag-megalith/codag-cli/internal/ui"
)

// silentErr wraps an error that has already been printed to the user.
type silentErr struct{ err error }

func (e *silentErr) Error() string { return e.err.Error() }
func (e *silentErr) Unwrap() error { return e.err }

// IsSilent returns true if the error has already been displayed.
func IsSilent(err error) bool {
	var s *silentErr
	return errors.As(err, &s)
}

// silent marks an error as already printed.
func silent(err error) error { return &silentErr{err} }

// handleAPIError formats API errors for display and returns a silent error.
func handleAPIError(err error, server string) error {
	var apiErr *api.APIError
	if errors.As(err, &apiErr) {
		switch apiErr.StatusCode {
		case 401:
			ui.Error("Invalid or expired token. Run: codag login")
		default:
			ui.Error(fmt.Sprintf("Error %d: %s", apiErr.StatusCode, apiErr.Detail))
		}
	} else {
		ui.Error(fmt.Sprintf("Cannot connect to %s", server))
		fmt.Fprintln(os.Stderr, "  Check your connection or try again later.")
	}
	return silent(err)
}
