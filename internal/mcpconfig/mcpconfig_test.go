package mcpconfig

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteNew(t *testing.T) {
	dir := t.TempDir()
	action, err := Write(dir, "https://api.codag.ai")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if action != "created" {
		t.Fatalf("expected action=created, got %s", action)
	}

	data, _ := os.ReadFile(filepath.Join(dir, ".mcp.json"))
	var config map[string]interface{}
	json.Unmarshal(data, &config)

	servers := config["mcpServers"].(map[string]interface{})
	codag := servers["codag"].(map[string]interface{})
	if codag["command"] != "codag" {
		t.Fatalf("expected command=codag, got %v", codag["command"])
	}
	args := codag["args"].([]interface{})
	if args[0] != "mcp" || args[1] != "serve" || args[2] != "." {
		t.Fatalf("expected args=[mcp serve .], got %v", args)
	}
	env := codag["env"].(map[string]interface{})
	if env["CODAG_URL"] != "https://api.codag.ai" {
		t.Fatalf("expected https://api.codag.ai, got %v", env["CODAG_URL"])
	}
}

func TestWriteUnchanged(t *testing.T) {
	dir := t.TempDir()
	Write(dir, "https://api.codag.ai")

	action, err := Write(dir, "https://api.codag.ai")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if action != "unchanged" {
		t.Fatalf("expected action=unchanged, got %s", action)
	}
}

func TestWriteUpdate(t *testing.T) {
	dir := t.TempDir()
	Write(dir, "https://api.codag.ai")

	action, err := Write(dir, "http://localhost:8000")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if action != "updated" {
		t.Fatalf("expected action=updated, got %s", action)
	}

	data, _ := os.ReadFile(filepath.Join(dir, ".mcp.json"))
	var config map[string]interface{}
	json.Unmarshal(data, &config)
	servers := config["mcpServers"].(map[string]interface{})
	codag := servers["codag"].(map[string]interface{})
	env := codag["env"].(map[string]interface{})
	if env["CODAG_URL"] != "http://localhost:8000" {
		t.Fatalf("expected http://localhost:8000, got %v", env["CODAG_URL"])
	}
}

func TestWriteMerge(t *testing.T) {
	dir := t.TempDir()
	existing := `{
  "mcpServers": {
    "other-tool": {
      "command": "npx",
      "args": ["-y", "other-tool"]
    }
  }
}`
	os.WriteFile(filepath.Join(dir, ".mcp.json"), []byte(existing), 0644)

	action, err := Write(dir, "https://api.codag.ai")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if action != "updated" {
		t.Fatalf("expected action=updated, got %s", action)
	}

	data, _ := os.ReadFile(filepath.Join(dir, ".mcp.json"))
	var config map[string]interface{}
	json.Unmarshal(data, &config)
	servers := config["mcpServers"].(map[string]interface{})

	if _, ok := servers["other-tool"]; !ok {
		t.Fatal("other-tool was not preserved")
	}
	if _, ok := servers["codag"]; !ok {
		t.Fatal("codag was not added")
	}
}

func TestWriteMalformed(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, ".mcp.json"), []byte(`{invalid json`), 0644)

	action, err := Write(dir, "https://api.codag.ai")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if action != "created" {
		t.Fatalf("expected action=created (fresh after malformed), got %s", action)
	}

	if _, err := os.Stat(filepath.Join(dir, ".mcp.json.bak")); err != nil {
		t.Fatal("backup file should exist")
	}

	data, _ := os.ReadFile(filepath.Join(dir, ".mcp.json"))
	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatalf("new file should be valid JSON: %v", err)
	}
}

// --- WriteAll tests ---

func TestWriteAllRootOnly(t *testing.T) {
	dir := t.TempDir()
	results := WriteAll(dir, "https://api.codag.ai")

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Path != ".mcp.json" {
		t.Fatalf("expected .mcp.json, got %s", results[0].Path)
	}
	if results[0].Action != "created" {
		t.Fatalf("expected created, got %s", results[0].Action)
	}
}

func TestWriteAllWithVSCode(t *testing.T) {
	dir := t.TempDir()
	os.Mkdir(filepath.Join(dir, ".vscode"), 0755)

	results := WriteAll(dir, "https://api.codag.ai")

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	// Check .vscode/mcp.json uses "servers" key
	data, err := os.ReadFile(filepath.Join(dir, ".vscode", "mcp.json"))
	if err != nil {
		t.Fatalf("failed to read .vscode/mcp.json: %v", err)
	}
	var config map[string]interface{}
	json.Unmarshal(data, &config)

	if _, ok := config["servers"]; !ok {
		t.Fatal("expected 'servers' key in .vscode/mcp.json")
	}
	if _, ok := config["mcpServers"]; ok {
		t.Fatal("should NOT have 'mcpServers' key in .vscode/mcp.json")
	}

	servers := config["servers"].(map[string]interface{})
	if _, ok := servers["codag"]; !ok {
		t.Fatal("codag entry missing from .vscode/mcp.json")
	}
}

func TestWriteAllWithCodex(t *testing.T) {
	dir := t.TempDir()
	os.Mkdir(filepath.Join(dir, ".codex"), 0755)

	results := WriteAll(dir, "https://api.codag.ai")

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	data, err := os.ReadFile(filepath.Join(dir, ".codex", "config.toml"))
	if err != nil {
		t.Fatalf("failed to read .codex/config.toml: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "[mcp_servers.codag]") {
		t.Fatal("expected [mcp_servers.codag] section")
	}
	if !strings.Contains(content, `command = "codag"`) {
		t.Fatal("expected command = codag")
	}
	if !strings.Contains(content, `CODAG_URL = "https://api.codag.ai"`) {
		t.Fatal("expected CODAG_URL")
	}
}

func TestWriteAllWithAllEditors(t *testing.T) {
	dir := t.TempDir()
	os.Mkdir(filepath.Join(dir, ".vscode"), 0755)
	os.Mkdir(filepath.Join(dir, ".codex"), 0755)

	results := WriteAll(dir, "https://api.codag.ai")

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	editors := map[string]bool{}
	for _, r := range results {
		editors[r.Path] = true
	}
	if !editors[".mcp.json"] {
		t.Fatal("missing .mcp.json")
	}
	if !editors[".vscode/mcp.json"] {
		t.Fatal("missing .vscode/mcp.json")
	}
	if !editors[".codex/config.toml"] {
		t.Fatal("missing .codex/config.toml")
	}
}

func TestCodexTOMLUpdate(t *testing.T) {
	dir := t.TempDir()
	os.Mkdir(filepath.Join(dir, ".codex"), 0755)

	// Write initial
	WriteAll(dir, "https://api.codag.ai")

	// Update with new URL
	results := WriteAll(dir, "http://localhost:8000")

	for _, r := range results {
		if r.Path == ".codex/config.toml" {
			if r.Action != "updated" {
				t.Fatalf("expected updated, got %s", r.Action)
			}
		}
	}

	data, _ := os.ReadFile(filepath.Join(dir, ".codex", "config.toml"))
	if !strings.Contains(string(data), "http://localhost:8000") {
		t.Fatal("expected updated URL in config.toml")
	}
	if strings.Contains(string(data), "https://api.codag.ai") {
		t.Fatal("old URL should be gone")
	}
}

func TestCodexTOMLUnchanged(t *testing.T) {
	dir := t.TempDir()
	os.Mkdir(filepath.Join(dir, ".codex"), 0755)

	WriteAll(dir, "https://api.codag.ai")
	results := WriteAll(dir, "https://api.codag.ai")

	for _, r := range results {
		if r.Path == ".codex/config.toml" {
			if r.Action != "unchanged" {
				t.Fatalf("expected unchanged, got %s", r.Action)
			}
		}
	}
}

func TestVSCodeMergesExisting(t *testing.T) {
	dir := t.TempDir()
	os.Mkdir(filepath.Join(dir, ".vscode"), 0755)

	existing := `{
  "servers": {
    "other-tool": {
      "command": "npx",
      "args": ["-y", "other-tool"]
    }
  }
}`
	os.WriteFile(filepath.Join(dir, ".vscode", "mcp.json"), []byte(existing), 0644)

	WriteAll(dir, "https://api.codag.ai")

	data, _ := os.ReadFile(filepath.Join(dir, ".vscode", "mcp.json"))
	var config map[string]interface{}
	json.Unmarshal(data, &config)

	servers := config["servers"].(map[string]interface{})
	if _, ok := servers["other-tool"]; !ok {
		t.Fatal("other-tool was not preserved in .vscode/mcp.json")
	}
	if _, ok := servers["codag"]; !ok {
		t.Fatal("codag was not added to .vscode/mcp.json")
	}
}
