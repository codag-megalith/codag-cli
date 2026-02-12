package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var (
	CodagHome string
	EnvFile   string
)

func init() {
	home := os.Getenv("CODAG_HOME")
	if home == "" {
		userHome, err := os.UserHomeDir()
		if err != nil {
			userHome = "."
		}
		home = filepath.Join(userHome, ".codag")
	}
	CodagHome = home
	EnvFile = filepath.Join(CodagHome, ".env")
}

// LoadEnv reads ~/.codag/.env into os.Environ.
// OS env vars take precedence (matching Python CLI behavior).
func LoadEnv() {
	f, err := os.Open(EnvFile)
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" {
			continue
		}
		// Only set if not already in environment
		if _, exists := os.LookupEnv(key); !exists {
			os.Setenv(key, value)
		}
	}
}

// GetToken returns the GITHUB_TOKEN from environment.
func GetToken() string {
	return os.Getenv("GITHUB_TOKEN")
}

// RequireToken returns the GITHUB_TOKEN, prompting the user if missing.
func RequireToken() (string, error) {
	LoadEnv()
	token := GetToken()
	if token != "" {
		return token, nil
	}

	fmt.Println("Codag needs a GitHub token to read your repo's PR history.")
	fmt.Println()
	fmt.Println("  Create one at: https://github.com/settings/tokens")
	fmt.Println("  Required scope: repo (or public_repo for public repos)")
	fmt.Println()
	fmt.Print("  Enter GITHUB_TOKEN: ")

	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		return "", fmt.Errorf("no input")
	}
	token = strings.TrimSpace(scanner.Text())
	if token == "" {
		return "", fmt.Errorf("no token provided")
	}

	if err := SaveEnvVar("GITHUB_TOKEN", token); err != nil {
		return "", fmt.Errorf("saving token: %w", err)
	}
	fmt.Printf("  Saved to %s\n\n", EnvFile)
	return token, nil
}

// SaveEnvVar writes or updates a key in ~/.codag/.env.
func SaveEnvVar(key, value string) error {
	if err := os.MkdirAll(CodagHome, 0755); err != nil {
		return fmt.Errorf("creating %s: %w", CodagHome, err)
	}

	var lines []string
	replaced := false

	data, err := os.ReadFile(EnvFile)
	if err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), key+"=") {
				lines = append(lines, key+"="+value)
				replaced = true
			} else {
				lines = append(lines, line)
			}
		}
	}

	if !replaced {
		lines = append(lines, key+"="+value)
	}

	// Clean up trailing empty lines
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}

	content := strings.Join(lines, "\n") + "\n"
	if err := os.WriteFile(EnvFile, []byte(content), 0600); err != nil {
		return fmt.Errorf("writing %s: %w", EnvFile, err)
	}

	os.Setenv(key, value)
	return nil
}

// GetServerURL returns CODAG_SERVER_URL from environment, or empty string.
func GetServerURL() string {
	return os.Getenv("CODAG_SERVER_URL")
}
