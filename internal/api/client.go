package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const DefaultServer = "https://api.codag.ai"

type Client struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client
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
		BaseURL: baseURL,
		Token:   token,
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
	url := c.BaseURL + path

	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshaling request: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("cannot connect to %s: %w", c.BaseURL, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode >= 400 {
		detail := string(respBody)
		// Try to parse {"detail": "..."} from error response
		var errResp struct {
			Detail string `json:"detail"`
		}
		if json.Unmarshal(respBody, &errResp) == nil && errResp.Detail != "" {
			detail = errResp.Detail
		}
		return nil, &APIError{StatusCode: resp.StatusCode, Detail: detail}
	}

	return respBody, nil
}
