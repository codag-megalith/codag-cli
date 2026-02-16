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

// GetAccessToken returns the Codag JWT access token from environment.
func GetAccessToken() string {
	return os.Getenv("CODAG_ACCESS_TOKEN")
}

// GetRefreshToken returns the Codag refresh token from environment.
func GetRefreshToken() string {
	return os.Getenv("CODAG_REFRESH_TOKEN")
}

// GetToken returns the Codag access token.
func GetToken() string {
	return GetAccessToken()
}

// SaveTokens saves access and refresh tokens to ~/.codag/.env.
func SaveTokens(accessToken, refreshToken string) error {
	if err := SaveEnvVar("CODAG_ACCESS_TOKEN", accessToken); err != nil {
		return err
	}
	return SaveEnvVar("CODAG_REFRESH_TOKEN", refreshToken)
}

// ClearTokens removes Codag tokens from ~/.codag/.env.
func ClearTokens() error {
	if err := RemoveEnvVar("CODAG_ACCESS_TOKEN"); err != nil {
		return err
	}
	return RemoveEnvVar("CODAG_REFRESH_TOKEN")
}

// HasAuth returns true if the user has auth configured.
func HasAuth() bool {
	LoadEnv()
	return GetAccessToken() != ""
}

// RequireAuth returns the access token, or an error if not logged in.
func RequireAuth() (string, error) {
	LoadEnv()

	if t := GetAccessToken(); t != "" {
		return t, nil
	}

	return "", fmt.Errorf("not logged in — run: codag login")
}

// SaveEnvVar writes or updates a key in ~/.codag/.env.
func SaveEnvVar(key, value string) error {
	if err := os.MkdirAll(CodagHome, 0700); err != nil {
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

// RemoveEnvVar removes a key from ~/.codag/.env.
func RemoveEnvVar(key string) error {
	data, err := os.ReadFile(EnvFile)
	if err != nil {
		return nil // file doesn't exist, nothing to remove
	}

	var lines []string
	for _, line := range strings.Split(string(data), "\n") {
		if !strings.HasPrefix(strings.TrimSpace(line), key+"=") {
			lines = append(lines, line)
		}
	}

	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}

	content := strings.Join(lines, "\n") + "\n"
	if err := os.WriteFile(EnvFile, []byte(content), 0600); err != nil {
		return fmt.Errorf("writing %s: %w", EnvFile, err)
	}

	os.Unsetenv(key)
	return nil
}

// GetServerURL returns the API server URL from environment, or empty string.
// Checks CODAG_SERVER_URL first, then CODAG_URL for compatibility with .mcp.json.
func GetServerURL() string {
	if s := os.Getenv("CODAG_SERVER_URL"); s != "" {
		return s
	}
	return os.Getenv("CODAG_URL")
}
