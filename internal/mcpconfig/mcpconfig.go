package mcpconfig

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Result describes what happened when writing a config file.
type Result struct {
	Editor string // "claude/cursor", "vscode", "codex"
	Path   string // relative path from repo root
	Action string // "created", "updated", "unchanged"
}

// CodagEntry returns the MCP server configuration for Codag.
func CodagEntry(serverURL string) map[string]interface{} {
	return map[string]interface{}{
		"command": "codag",
		"args":    []string{"mcp", "serve", "."},
		"env": map[string]string{
			"CODAG_URL": serverURL,
		},
	}
}

// WriteAll writes MCP configs for all detected editors.
// Always writes .mcp.json (Claude Code + Cursor).
// Writes .vscode/mcp.json if .vscode/ exists.
// Writes .codex/config.toml if .codex/ exists.
func WriteAll(dir string, serverURL string) []Result {
	var results []Result

	// Always: .mcp.json at root (Claude Code + Cursor)
	action, err := writeRootMCP(dir, serverURL)
	if err == nil {
		results = append(results, Result{Editor: "Claude Code / Cursor", Path: ".mcp.json", Action: action})
	}

	// If .vscode/ exists: .vscode/mcp.json
	if dirExists(filepath.Join(dir, ".vscode")) {
		action, err := writeVSCodeMCP(dir, serverURL)
		if err == nil {
			results = append(results, Result{Editor: "VS Code", Path: ".vscode/mcp.json", Action: action})
		}
	}

	// If .codex/ exists: .codex/config.toml
	if dirExists(filepath.Join(dir, ".codex")) {
		action, err := writeCodexTOML(dir, serverURL)
		if err == nil {
			results = append(results, Result{Editor: "Codex", Path: ".codex/config.toml", Action: action})
		}
	}

	return results
}

// Write creates or updates .mcp.json in the given directory (backward compat).
// Returns ("created"|"updated"|"unchanged", error).
func Write(dir string, serverURL string) (string, error) {
	return writeRootMCP(dir, serverURL)
}

// writeRootMCP writes .mcp.json with mcpServers key (Claude Code + Cursor).
func writeRootMCP(dir string, serverURL string) (string, error) {
	return writeJSONConfig(filepath.Join(dir, ".mcp.json"), "mcpServers", serverURL)
}

// writeVSCodeMCP writes .vscode/mcp.json with servers key (VS Code Copilot).
func writeVSCodeMCP(dir string, serverURL string) (string, error) {
	return writeJSONConfig(filepath.Join(dir, ".vscode", "mcp.json"), "servers", serverURL)
}

// writeJSONConfig writes a JSON MCP config file with the given wrapper key.
func writeJSONConfig(path string, serversKey string, serverURL string) (string, error) {
	entry := CodagEntry(serverURL)

	var config map[string]interface{}
	action := "created"

	data, err := os.ReadFile(path)
	if err == nil {
		if err := json.Unmarshal(data, &config); err != nil {
			backupPath := path + ".bak"
			os.Rename(path, backupPath)
			config = nil
		} else {
			action = "updated"
		}
	}

	if config == nil {
		config = make(map[string]interface{})
	}

	servers, ok := config[serversKey]
	if !ok {
		servers = make(map[string]interface{})
		config[serversKey] = servers
	}

	serversMap, ok := servers.(map[string]interface{})
	if !ok {
		serversMap = make(map[string]interface{})
		config[serversKey] = serversMap
	}

	if existing, ok := serversMap["codag"]; ok {
		existingJSON, _ := json.Marshal(existing)
		newJSON, _ := json.Marshal(entry)
		if string(existingJSON) == string(newJSON) {
			return "unchanged", nil
		}
	}

	serversMap["codag"] = entry

	output, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshaling config: %w", err)
	}
	output = append(output, '\n')

	if err := os.WriteFile(path, output, 0644); err != nil {
		return "", fmt.Errorf("%w", err)
	}

	return action, nil
}

// writeCodexTOML writes .codex/config.toml with [mcp_servers.codag] section.
func writeCodexTOML(dir string, serverURL string) (string, error) {
	path := filepath.Join(dir, ".codex", "config.toml")

	// The TOML section we want
	section := fmt.Sprintf(`[mcp_servers.codag]
command = "codag"
args = ["mcp", "serve", "."]

[mcp_servers.codag.env]
CODAG_URL = "%s"`, serverURL)

	data, err := os.ReadFile(path)
	if err != nil {
		// File doesn't exist — create it
		if err := os.WriteFile(path, []byte(section+"\n"), 0644); err != nil {
			return "", err
		}
		return "created", nil
	}

	content := string(data)

	// Check if codag section already exists
	if strings.Contains(content, "[mcp_servers.codag]") {
		// Check if it's identical
		if strings.Contains(content, section) {
			return "unchanged", nil
		}
		// Replace existing codag section — find start and end
		// Remove old section and append new one
		content = removeTOMLSection(content, "mcp_servers.codag")
		content = strings.TrimRight(content, "\n") + "\n\n" + section + "\n"
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return "", err
		}
		return "updated", nil
	}

	// Append to existing file
	content = strings.TrimRight(content, "\n") + "\n\n" + section + "\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return "", err
	}
	return "updated", nil
}

// removeTOMLSection removes all lines belonging to a TOML section and its subsections.
func removeTOMLSection(content string, sectionPrefix string) string {
	lines := strings.Split(content, "\n")
	var result []string
	inSection := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "["+sectionPrefix+"]") || strings.HasPrefix(trimmed, "["+sectionPrefix+".") {
			inSection = true
			continue
		}
		if inSection && strings.HasPrefix(trimmed, "[") {
			// New section that's not ours — stop removing
			inSection = false
		}
		if !inSection {
			result = append(result, line)
		}
	}

	return strings.Join(result, "\n")
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
