package mcp

import (
	"context"
	"encoding/json"
	"os"

	gomcp "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func Serve(workspacePath, serverURL, version string) error {
	token := os.Getenv("CODAG_TOKEN")
	if token == "" {
		token = os.Getenv("GITHUB_TOKEN")
	}

	client := NewClient(serverURL, token, workspacePath)
	client.CheckAvailability()

	s := server.NewMCPServer(
		"codag",
		version,
		server.WithToolCapabilities(false),
	)

	s.AddTool(briefTool(), briefHandler(client))
	s.AddTool(checkTool(), checkHandler(client))

	return server.ServeStdio(s)
}

func briefTool() gomcp.Tool {
	return gomcp.NewTool("codag_brief",
		gomcp.WithDescription("Get organizational memory signals for files you're about to modify. Returns danger zones, rejected approaches, co-change partners, and churn heat. Call this before editing files."),
		gomcp.WithArray("files",
			gomcp.Required(),
			gomcp.Description("File paths relative to the repo root"),
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

func checkTool() gomcp.Tool {
	return gomcp.NewTool("codag_check",
		gomcp.WithDescription("Check if an approach or pattern was previously tried and rejected in this codebase. Call this before implementing non-trivial changes to avoid repeating past mistakes."),
		gomcp.WithString("description",
			gomcp.Required(),
			gomcp.Description("Description of the approach or change you're planning"),
		),
	)
}

func checkHandler(client *Client) server.ToolHandlerFunc {
	return func(ctx context.Context, req gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
		desc := req.GetString("description", "")
		if desc == "" {
			return gomcp.NewToolResultError("missing required parameter: description"), nil
		}

		result, err := client.Check(desc)
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
