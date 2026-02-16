package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/codag-megalith/codag-cli/internal/config"
)

const DefaultServer = "https://api.codag.ai"

type Client struct {
	BaseURL      string
	Token        string
	RefreshToken string
	HTTPClient   *http.Client
}

type RepoResponse struct {
	ID            int     `json:"id"`
	Name          string  `json:"name"`
	Owner         string  `json:"owner"`
	GithubURL     string  `json:"github_url"`
	LastIndexedAt *string `json:"last_indexed_at"`
}

type BackfillResponse struct {
	RepoID  int    `json:"repo_id"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

type StatsResponse struct {
	RepoID           int  `json:"repo_id"`
	PRsIndexed       int  `json:"prs_indexed"`
	FilesWithSignals int  `json:"files_with_signals"`
	TotalSignals     int  `json:"total_signals"`
	DangerSignals    int  `json:"danger_signals"`
	Indexing         bool `json:"indexing"`
}

type APIError struct {
	StatusCode int
	Detail     string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("Error %d: %s", e.StatusCode, e.Detail)
}

func NewClient(baseURL, token string) *Client {
	return &Client{
		BaseURL:      baseURL,
		Token:        token,
		RefreshToken: config.GetRefreshToken(),
		HTTPClient: &http.Client{
			Timeout: 600 * time.Second,
		},
	}
}

func (c *Client) RegisterRepo(githubURL string) (*RepoResponse, error) {
	body := map[string]string{"github_url": githubURL}
	data, err := c.do("POST", "/api/repos", body)
	if err != nil {
		return nil, err
	}
	var resp RepoResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}
	return &resp, nil
}

func (c *Client) TriggerBackfill(repoID int, maxPRs *int) (*BackfillResponse, error) {
	path := fmt.Sprintf("/api/repos/%d/backfill", repoID)
	if maxPRs != nil {
		path += fmt.Sprintf("?max_prs=%d", *maxPRs)
	}
	data, err := c.do("POST", path, nil)
	if err != nil {
		return nil, err
	}
	var resp BackfillResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}
	return &resp, nil
}

func (c *Client) ListRepos() ([]RepoResponse, error) {
	data, err := c.do("GET", "/api/repos", nil)
	if err != nil {
		return nil, err
	}
	var resp []RepoResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}
	return resp, nil
}

type WebhookResponse struct {
	Status    string `json:"status"`
	WebhookID int    `json:"webhook_id,omitempty"`
	Message   string `json:"message,omitempty"`
}

func (c *Client) SetupWebhook(repoID int) (*WebhookResponse, error) {
	path := fmt.Sprintf("/api/repos/%d/setup-webhook", repoID)
	data, err := c.do("POST", path, nil)
	if err != nil {
		return nil, err
	}
	var resp WebhookResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}
	return &resp, nil
}

func (c *Client) GetStats(repoID int) (*StatsResponse, error) {
	path := fmt.Sprintf("/api/stats?repo=%d", repoID)
	data, err := c.do("GET", path, nil)
	if err != nil {
		return nil, err
	}
	var resp StatsResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}
	return &resp, nil
}

type MeResponse struct {
	User struct {
		GithubLogin string `json:"github_login"`
		Email       string `json:"email"`
		CreatedAt   string `json:"created_at"`
	} `json:"user"`
	Subscription *struct {
		Tier              string `json:"tier"`
		Status            string `json:"status"`
		BillingInterval   string `json:"billing_interval"`
		CurrentPeriodEnd  string `json:"current_period_end"`
		CancelAtPeriodEnd bool   `json:"cancel_at_period_end"`
	} `json:"subscription"`
	Repos []RepoResponse `json:"repos"`
	Orgs  []struct {
		Name string `json:"name"`
		Slug string `json:"slug"`
		Role string `json:"role"`
		Tier string `json:"tier"`
	} `json:"orgs"`
}

func (c *Client) GetMe() (*MeResponse, error) {
	data, err := c.do("GET", "/api/console/me", nil)
	if err != nil {
		return nil, err
	}
	var resp MeResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}
	return &resp, nil
}

func (c *Client) do(method, path string, body interface{}) ([]byte, error) {
	data, statusCode, err := c.doRaw(method, path, body)
	if err != nil {
		return nil, err
	}

	// On 401, try to refresh tokens and retry once
	if statusCode == 401 && c.RefreshToken != "" {
		if c.tryRefresh() {
			data, statusCode, err = c.doRaw(method, path, body)
			if err != nil {
				return nil, err
			}
		}
	}

	if statusCode >= 400 {
		var detail string
		var errResp struct {
			Detail string `json:"detail"`
		}
		if json.Unmarshal(data, &errResp) == nil && errResp.Detail != "" {
			detail = errResp.Detail
		} else {
			// Raw body — truncate to avoid leaking proxy HTML pages
			detail = string(data)
			if len(detail) > 200 {
				detail = detail[:200] + "…"
			}
		}
		return nil, &APIError{StatusCode: statusCode, Detail: detail}
	}

	return data, nil
}

func (c *Client) doRaw(method, path string, body interface{}) ([]byte, int, error) {
	url := c.BaseURL + path

	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, 0, fmt.Errorf("marshaling request: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, 0, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("cannot connect to %s: %w", c.BaseURL, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, fmt.Errorf("reading response: %w", err)
	}

	return respBody, resp.StatusCode, nil
}

// tryRefresh attempts to refresh the access token using the refresh token.
// Returns true if refresh succeeded and tokens were updated.
func (c *Client) tryRefresh() bool {
	body, _ := json.Marshal(map[string]string{"refresh_token": c.RefreshToken})

	req, err := http.NewRequest("POST", c.BaseURL+"/api/auth/refresh", bytes.NewReader(body))
	if err != nil {
		return false
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return false
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return false
	}

	// Update client state
	c.Token = tokenResp.AccessToken
	c.RefreshToken = tokenResp.RefreshToken

	// Persist to disk
	if err := config.SaveTokens(tokenResp.AccessToken, tokenResp.RefreshToken); err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not save refreshed tokens: %s\n", err)
	}

	return true
}
