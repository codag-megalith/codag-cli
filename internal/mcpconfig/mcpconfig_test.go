package mcpconfig

import (
	"encoding/json"
	"os"
	"path/filepath"
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

	data, _ := os.ReadFile(filepath.Join(dir, FileName))
	var config map[string]interface{}
	json.Unmarshal(data, &config)

	servers := config["mcpServers"].(map[string]interface{})
	codag := servers["codag"].(map[string]interface{})
	if codag["command"] != "npx" {
		t.Fatalf("expected command=npx, got %v", codag["command"])
	}
	args := codag["args"].([]interface{})
	if args[1] != "@codag/mcp-server" {
		t.Fatalf("expected @codag/mcp-server, got %v", args[1])
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

	data, _ := os.ReadFile(filepath.Join(dir, FileName))
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
	os.WriteFile(filepath.Join(dir, FileName), []byte(existing), 0644)

	action, err := Write(dir, "https://api.codag.ai")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if action != "updated" {
		t.Fatalf("expected action=updated, got %s", action)
	}

	data, _ := os.ReadFile(filepath.Join(dir, FileName))
	var config map[string]interface{}
	json.Unmarshal(data, &config)
	servers := config["mcpServers"].(map[string]interface{})

	// other-tool preserved
	if _, ok := servers["other-tool"]; !ok {
		t.Fatal("other-tool was not preserved")
	}
	// codag added
	if _, ok := servers["codag"]; !ok {
		t.Fatal("codag was not added")
	}
}

func TestWriteMalformed(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, FileName), []byte(`{invalid json`), 0644)

	action, err := Write(dir, "https://api.codag.ai")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if action != "created" {
		t.Fatalf("expected action=created (fresh after malformed), got %s", action)
	}

	// Backup should exist
	if _, err := os.Stat(filepath.Join(dir, FileName+".bak")); err != nil {
		t.Fatal("backup file should exist")
	}

	// New file should be valid JSON
	data, _ := os.ReadFile(filepath.Join(dir, FileName))
	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatalf("new file should be valid JSON: %v", err)
	}
}
