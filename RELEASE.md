# Release Setup

## Prerequisites

1. **Create GitHub repo** — `codag-megalith/codag-cli`
2. **Create Homebrew tap repo** — `codag-megalith/homebrew-tap` (empty repo, GoReleaser will push to it)
3. **Create a GitHub PAT** with `repo` scope (needed for GoReleaser to push to the tap repo)
4. **Add the PAT as a repo secret** — go to `codag-megalith/codag-cli` → Settings → Secrets → add `HOMEBREW_TAP_TOKEN`

## First Release

```bash
git init && git add . && git commit -m "initial commit"
git remote add origin git@github.com:codag-megalith/codag-cli.git
git push -u origin main
git tag v0.1.0
git push origin v0.1.0
```

The tag push triggers the GitHub Action which runs GoReleaser, which:
- Builds binaries for linux/darwin/windows (amd64 + arm64)
- Creates a GitHub Release with the binaries attached
- Pushes the Homebrew formula to `codag-megalith/homebrew-tap`

## What Still Needs to Be Created

- [ ] `.github/workflows/release.yml` — GitHub Action to run GoReleaser on tag push
- [ ] `install.sh` — curl-able install script, host at `https://codag.ai/install.sh`

## Once Released, Users Can Install Via

```bash
# Option 1: curl pipe
curl -fsSL https://codag.ai/install.sh | sh

# Option 2: Homebrew
brew install codag-megalith/tap/codag

# Option 3: Go
go install github.com/codag-megalith/codag-cli@latest
```

## Upgrading

`codag upgrade` — self-updates the binary from GitHub Releases.

Users are also automatically notified once per day if a newer version exists.
