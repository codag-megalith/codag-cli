package mcpconfig

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const FileName = ".mcp.json"

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

// Write creates or updates .mcp.json in the given directory.
// Returns ("created"|"updated"|"unchanged", error).
func Write(dir string, serverURL string) (string, error) {
	path := filepath.Join(dir, FileName)
	entry := CodagEntry(serverURL)

	// Read existing file
	var config map[string]interface{}
	action := "created"

	data, err := os.ReadFile(path)
	if err == nil {
		if err := json.Unmarshal(data, &config); err != nil {
			// Malformed JSON — back up and start fresh
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

	// Ensure mcpServers key exists
	servers, ok := config["mcpServers"]
	if !ok {
		servers = make(map[string]interface{})
		config["mcpServers"] = servers
	}

	serversMap, ok := servers.(map[string]interface{})
	if !ok {
		serversMap = make(map[string]interface{})
		config["mcpServers"] = serversMap
	}

	// Check if already exists and is identical
	if existing, ok := serversMap["codag"]; ok {
		existingJSON, _ := json.Marshal(existing)
		newJSON, _ := json.Marshal(entry)
		if string(existingJSON) == string(newJSON) {
			return "unchanged", nil
		}
		if action == "created" {
			action = "updated"
		}
	} else if action == "updated" {
		action = "updated" // file existed but no codag key — "added codag server"
	}

	serversMap["codag"] = entry

	// Write back with 2-space indent (matching JS conventions)
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
