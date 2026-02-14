package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
		detail := string(data)
		var errResp struct {
			Detail string `json:"detail"`
		}
		if json.Unmarshal(data, &errResp) == nil && errResp.Detail != "" {
			detail = errResp.Detail
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
	_ = config.SaveTokens(tokenResp.AccessToken, tokenResp.RefreshToken)

	return true
}
