package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"runtime"
	"time"

	"github.com/codag-megalith/codag-cli/internal/api"
	"github.com/codag-megalith/codag-cli/internal/config"
	"github.com/codag-megalith/codag-cli/internal/ui"
	"github.com/spf13/cobra"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with Codag",
	RunE: func(cmd *cobra.Command, args []string) error {
		server := resolveServer(cmd)

		// Check if existing session is still valid
		if config.HasAuth() {
			client := api.NewClient(server, config.GetAccessToken())
			_, err := client.ListRepos()
			if err == nil {
				fmt.Print("Already logged in. Re-authenticate? [y/N] ")
				var answer string
				fmt.Scanln(&answer)
				if answer != "y" && answer != "Y" {
					ui.Info("Kept existing session.")
					return nil
				}
				fmt.Println()
			} else {
				// Token expired or invalid — clear and re-auth
				ui.Warn("Session expired. Logging in again...")
				config.ClearTokens()
				fmt.Println()
			}
		}

		// Device code flow (requires Brain server with JWT configured)
		isDev, _ := cmd.Flags().GetBool("dev")
		err := deviceCodeLogin(server, isDev)
		if err != nil {
			if apiErr, ok := err.(*api.APIError); ok && apiErr.StatusCode == 501 {
				ui.Error("Server does not have JWT auth configured. Contact your admin.")
				return fmt.Errorf("server JWT not configured")
			}
			return err
		}

		return nil
	},
}

func init() {
	addServerFlag(loginCmd)
}

func deviceCodeLogin(serverURL string, isDev bool) error {
	httpClient := &http.Client{Timeout: 15 * time.Second}

	// Step 1: Request device code
	req, err := http.NewRequest("POST", serverURL+"/api/auth/device", nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		ui.Error(fmt.Sprintf("Cannot connect to %s", serverURL))
		return fmt.Errorf("connecting to server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		var errResp struct {
			Detail string `json:"detail"`
		}
		json.NewDecoder(resp.Body).Decode(&errResp)
		return &api.APIError{StatusCode: resp.StatusCode, Detail: errResp.Detail}
	}

	var deviceResp struct {
		DeviceCode      string `json:"device_code"`
		UserCode        string `json:"user_code"`
		VerificationURI string `json:"verification_uri"`
		ExpiresIn       int    `json:"expires_in"`
		Interval        int    `json:"interval"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&deviceResp); err != nil {
		return fmt.Errorf("parsing response: %w", err)
	}

	// Step 2: Display code and open browser
	verificationURI := deviceResp.VerificationURI
	if isDev {
		verificationURI = "http://localhost:3000/device"
	}
	fmt.Println()
	fmt.Printf("  Your code: %s\n", ui.Bold.Render(deviceResp.UserCode))
	fmt.Println()

	if err := openBrowser(verificationURI); err == nil {
		fmt.Println("  Browser opened. If it didn't open, visit:")
	} else {
		fmt.Println("  Open this URL in your browser:")
	}
	fmt.Printf("    %s\n", ui.Bold.Render(verificationURI))
	fmt.Println()

	// Step 3: Poll for authorization
	spinner := ui.NewSpinner("Waiting for authorization...")
	spinner.Start()
	defer spinner.Stop()

	interval := time.Duration(deviceResp.Interval) * time.Second
	if interval < 3*time.Second {
		interval = 5 * time.Second
	}
	deadline := time.Now().Add(time.Duration(deviceResp.ExpiresIn) * time.Second)

	for time.Now().Before(deadline) {
		time.Sleep(interval)

		tokenBody, _ := json.Marshal(map[string]string{
			"device_code": deviceResp.DeviceCode,
		})

		pollReq, _ := http.NewRequest("POST", serverURL+"/api/auth/device/token", bytes.NewReader(tokenBody))
		pollReq.Header.Set("Content-Type", "application/json")

		pollResp, err := httpClient.Do(pollReq)
		if err != nil {
			continue // network hiccup, retry
		}

		switch pollResp.StatusCode {
		case 428:
			// Still pending — keep polling
			pollResp.Body.Close()
			continue

		case 410:
			pollResp.Body.Close()
			spinner.Stop()
			ui.Error("Device code expired. Please try again.")
			return fmt.Errorf("device code expired")

		case 200:
			// Success!
			var tokenResp struct {
				AccessToken  string `json:"access_token"`
				RefreshToken string `json:"refresh_token"`
				User         *struct {
					GithubLogin string `json:"github_login"`
				} `json:"user"`
			}
			if err := json.NewDecoder(pollResp.Body).Decode(&tokenResp); err != nil {
				pollResp.Body.Close()
				return fmt.Errorf("parsing token response: %w", err)
			}
			pollResp.Body.Close()

			// Save tokens
			if err := config.SaveTokens(tokenResp.AccessToken, tokenResp.RefreshToken); err != nil {
				return fmt.Errorf("saving tokens: %w", err)
			}

			spinner.Stop()
			login := ""
			if tokenResp.User != nil {
				login = tokenResp.User.GithubLogin
			}
			if login != "" {
				ui.Success(fmt.Sprintf("Logged in as %s", login))
			} else {
				ui.Success("Logged in successfully")
			}
			fmt.Printf("  Tokens saved to %s\n", config.EnvFile)
			return nil

		default:
			pollResp.Body.Close()
			spinner.Stop()
			ui.Error(fmt.Sprintf("Unexpected response: %d", pollResp.StatusCode))
			return fmt.Errorf("unexpected status: %d", pollResp.StatusCode)
		}
	}

	spinner.Stop()
	ui.Error("Timed out waiting for authorization. Please try again.")
	return fmt.Errorf("authorization timed out")
}

func openBrowser(url string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", url).Start()
	case "linux":
		return exec.Command("xdg-open", url).Start()
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	default:
		return fmt.Errorf("unsupported platform")
	}
}

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Sign out and clear saved tokens",
	RunE: func(cmd *cobra.Command, args []string) error {
		server := resolveServer(cmd)

		// Revoke refresh token on server if possible
		if rt := config.GetRefreshToken(); rt != "" {
			httpClient := &http.Client{Timeout: 5 * time.Second}
			body, _ := json.Marshal(map[string]string{"refresh_token": rt})
			req, _ := http.NewRequest("POST", server+"/api/auth/logout", bytes.NewReader(body))
			if req != nil {
				req.Header.Set("Content-Type", "application/json")
				httpClient.Do(req) // best-effort, ignore errors
			}
		}

		// Clear local tokens
		if err := config.ClearTokens(); err != nil {
			ui.Warn(fmt.Sprintf("Could not clear tokens: %s", err))
		}

		// Also clear legacy token
		_ = config.RemoveEnvVar("GITHUB_TOKEN")

		ui.Success("Logged out.")
		return nil
	},
}

func init() {
	addServerFlag(logoutCmd)
}
