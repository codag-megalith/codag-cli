package mcp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

var sshRemoteRe = regexp.MustCompile(`^git@github\.com:(.+?)(?:\.git)?$`)

type Client struct {
	baseURL       string
	token         string
	repoID        int
	available     bool
	workspacePath string
}

type resolvedRepo struct {
	ID       int    `json:"id"`
	GithubURL string `json:"github_url"`
	Name     string `json:"name"`
	Owner    string `json:"owner"`
}

func NewClient(baseURL, token, workspacePath string) *Client {
	return &Client{
		baseURL:       strings.TrimRight(baseURL, "/"),
		token:         token,
		workspacePath: workspacePath,
	}
}

func (c *Client) CheckAvailability() bool {
	httpClient := &http.Client{Timeout: 3 * time.Second}

	// 1. Health check
	resp, err := httpClient.Get(c.baseURL + "/api/health")
	if err != nil || resp.StatusCode != http.StatusOK {
		return false
	}
	resp.Body.Close()

	// 2. Detect git remote
	githubURL := detectGitRemote(c.workspacePath)
	if githubURL == "" {
		return false
	}

	// 3. Resolve repo ID
	resolveURL, _ := url.Parse(c.baseURL + "/api/repos/resolve")
	q := resolveURL.Query()
	q.Set("github_url", githubURL)
	resolveURL.RawQuery = q.Encode()

	req, _ := http.NewRequest("GET", resolveURL.String(), nil)
	c.setHeaders(req)
	req.Header.Set("Content-Type", "application/json")

	httpClient.Timeout = 5 * time.Second
	resp, err = httpClient.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		return false
	}
	defer resp.Body.Close()

	var repo resolvedRepo
	if err := json.NewDecoder(resp.Body).Decode(&repo); err != nil {
		return false
	}

	c.repoID = repo.ID
	c.available = true
	return true
}

func (c *Client) Brief(files []string) (json.RawMessage, error) {
	if !c.available {
		return unavailableResponse()
	}
	body := map[string]interface{}{"repo": c.repoID, "files": files}
	return c.post("/api/brief", body)
}

func (c *Client) Check(description string) (json.RawMessage, error) {
	if !c.available {
		return unavailableResponse()
	}
	body := map[string]interface{}{"repo": c.repoID, "description": description}
	return c.post("/api/check", body)
}

func (c *Client) post(path string, body interface{}) (json.RawMessage, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	httpClient := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("POST", c.baseURL+path, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	c.setHeaders(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var raw json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return raw, fmt.Errorf("API returned %d", resp.StatusCode)
	}

	return raw, nil
}

func (c *Client) setHeaders(req *http.Request) {
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
}

func unavailableResponse() (json.RawMessage, error) {
	msg := map[string]string{
		"error":   "codag_unavailable",
		"message": "Codag is not connected for this repo. Run `codag init` in your repo first.",
	}
	data, _ := json.Marshal(msg)
	return data, nil
}

func detectGitRemote(workspacePath string) string {
	cmd := exec.Command("git", "remote", "get-url", "origin")
	cmd.Dir = workspacePath
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	remote := strings.TrimSpace(string(out))

	if m := sshRemoteRe.FindStringSubmatch(remote); m != nil {
		return "https://github.com/" + m[1]
	}
	if strings.Contains(remote, "github.com") {
		return strings.TrimSuffix(remote, ".git")
	}
	return ""
}
