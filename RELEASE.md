# Release Setup

## Prerequisites

1. **Create GitHub repo** — `codag-megalith/codag-cli` (public)

## DNS Setup

Point `codag.ai/install.sh` to the raw install script. Options:

- **Redirect**: `codag.ai/install.sh` → `https://raw.githubusercontent.com/codag-megalith/codag-cli/main/install.sh`
- **Or serve directly** from your site/CDN

## First Release

```bash
git init && git add . && git commit -m "initial commit"
git remote add origin git@github.com:codag-megalith/codag-cli.git
git push -u origin main
git tag v0.1.0
git push origin v0.1.0
```

The tag push triggers `.github/workflows/release.yml` → GoReleaser:
- Builds binaries for linux/darwin/windows (amd64 + arm64)
- Generates `checksums.txt` (SHA256)
- Creates a GitHub Release with binaries + checksums

## Users Install Via

```bash
curl -fsSL https://codag.ai/install.sh | bash
```

## Upgrading

`codag upgrade` — self-updates the binary from GitHub Releases.

Users are automatically notified once per day if a newer version exists.
