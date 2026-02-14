# CLAUDE.md — Codag CLI

## What This Is

Go CLI for Codag — **organizational memory for AI coding agents**. Handles user auth, repo registration, indexing, and runs the MCP server that feeds signals to coding agents.

## Architecture

```
codag login          → Device code OAuth (JWT)
codag init [url]     → Register repo + index + write .mcp.json
codag index          → Re-index a registered repo
codag status         → Show indexing stats
codag mcp serve .    → MCP server over stdio (spawned by agents via .mcp.json)
codag upgrade        → Self-update from GitHub Releases
codag logout         → Revoke tokens + clear local config
```

## Auth Flow

1. **Device code OAuth**: `codag login` → browser auth → JWT access + refresh tokens stored in `~/.codag/.env`
2. **Token refresh**: API client auto-refreshes on 401 via `/api/auth/refresh`, persists new tokens to disk

Auth token: `CODAG_ACCESS_TOKEN` (JWT issued by Brain)

## Key Files

| File | Purpose |
|------|---------|
| `cmd/root.go` | Root command, server resolution, update check hooks |
| `cmd/login.go` | Device code OAuth login + logout command |
| `cmd/init.go` | Repo registration, git remote detection, .mcp.json writing |
| `cmd/mcp.go` | `codag mcp serve` subcommand |
| `cmd/upgrade.go` | Self-update from GitHub Releases |
| `cmd/updatecheck.go` | Background version check (24h cache) |
| `internal/api/client.go` | HTTP client with auto token refresh on 401 |
| `internal/config/config.go` | Token management, `~/.codag/.env` read/write |
| `internal/mcp/server.go` | MCP server (mcp-go), exposes `codag_brief` + `codag_check` tools |
| `internal/mcp/client.go` | HTTP client for Brain API (brief, check), repo resolution from git remote |
| `internal/mcpconfig/mcpconfig.go` | .mcp.json writer (creates/updates/merges) |
| `internal/ui/` | Terminal UI helpers (spinner, colors, styled output) |

## Config

All config stored in `~/.codag/.env`:
```
CODAG_ACCESS_TOKEN=<jwt>
CODAG_REFRESH_TOKEN=<jwt>
```

Server URL resolution: `--server` flag > `CODAG_SERVER_URL` env > `CODAG_URL` env > `https://api.codag.ai`

## MCP Server

The `codag mcp serve .` command starts an MCP server over stdio using `mcp-go`. Two tools:

- **`codag_brief`** — Pre-computed danger signals for files. Agent calls before modifying files.
- **`codag_check`** — Check if an approach was previously rejected. Agent calls before architectural changes.

Both tools call the Brain API (`/api/brief`, `/api/check`) with the repo ID resolved from the workspace's git remote.

`.mcp.json` written by `codag init`:
```json
{
  "mcpServers": {
    "codag": {
      "command": "codag",
      "args": ["mcp", "serve", "."],
      "env": { "CODAG_URL": "https://api.codag.ai" }
    }
  }
}
```

## Build & Release

```bash
make build        # → bin/codag
make install      # → go install
make test         # → go test ./...
```

Release via GoReleaser (`.goreleaser.yml`): builds linux/darwin/windows (amd64+arm64), publishes to GitHub Releases + `codag-megalith/homebrew-tap`.

## Related Projects

- **codag-brain** (`../codag-brain`) — Python backend. Indexes PR history, extracts signals via Gemini Flash, serves via API.
- **codag-console** (`../codag-console`) — Next.js dashboard for monitoring signals and repos.
