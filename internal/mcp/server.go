package mcp

import (
	"context"
	"encoding/json"
	"os"

	gomcp "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func Serve(workspacePath, serverURL, version string) error {
	token := os.Getenv("CODAG_ACCESS_TOKEN")
	refreshToken := os.Getenv("CODAG_REFRESH_TOKEN")

	client := NewClient(serverURL, token, refreshToken, workspacePath)
	client.CheckAvailability()

	s := server.NewMCPServer(
		"codag",
		version,
		server.WithToolCapabilities(false),
	)

	s.AddTool(briefTool(), briefHandler(client))

	return server.ServeStdio(s)
}

func briefTool() gomcp.Tool {
	return gomcp.NewTool("codag_brief",
		gomcp.WithDescription("Get pre-computed danger signals, warnings, and patterns for files you're about to modify. Call this ONCE with all files before making changes. Returns ranked signals with inline context — no follow-up calls needed."),
		gomcp.WithArray("files",
			gomcp.Required(),
			gomcp.Description("File paths relative to repo root (e.g. ['src/main.py', 'src/utils.py'])"),
			gomcp.Items(map[string]any{"type": "string"}),
		),
	)
}

func briefHandler(client *Client) server.ToolHandlerFunc {
	return func(ctx context.Context, req gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
		filesRaw, ok := req.GetArguments()["files"]
		if !ok {
			return gomcp.NewToolResultError("missing required parameter: files"), nil
		}

		arr, ok := filesRaw.([]interface{})
		if !ok {
			return gomcp.NewToolResultError("files must be an array of strings"), nil
		}

		files := make([]string, 0, len(arr))
		for _, v := range arr {
			if s, ok := v.(string); ok {
				files = append(files, s)
			}
		}

		if len(files) == 0 {
			return gomcp.NewToolResultError("files array is empty"), nil
		}

		result, err := client.Brief(files)
		if err != nil {
			return gomcp.NewToolResultText(formatJSON(result)), nil
		}

		return gomcp.NewToolResultText(formatJSON(result)), nil
	}
}

func formatJSON(raw json.RawMessage) string {
	if raw == nil {
		return "{}"
	}
	var v interface{}
	if err := json.Unmarshal(raw, &v); err == nil {
		if b, err := json.MarshalIndent(v, "", "  "); err == nil {
			return string(b)
		}
	}
	return string(raw)
}
