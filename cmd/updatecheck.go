package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/codag-org/codag-cli/internal/config"
	"github.com/codag-org/codag-cli/internal/ui"
)

const checkInterval = 24 * time.Hour

type updateCache struct {
	CheckedAt     time.Time `json:"checked_at"`
	LatestVersion string    `json:"latest_version"`
}

var updateAvailable string // set by background check, read after command runs

func cacheFilePath() string {
	return filepath.Join(config.CodagHome, ".update-check")
}

// startUpdateCheck runs a non-blocking background version check.
// It writes results to updateAvailable for printUpdateNotice to consume.
func startUpdateCheck(done chan struct{}) {
	go func() {
		defer close(done)

		if Version == "dev" {
			return
		}

		// Check if we looked recently
		cache, err := readCache()
		if err == nil && time.Since(cache.CheckedAt) < checkInterval {
			current := strings.TrimPrefix(Version, "v")
			latest := strings.TrimPrefix(cache.LatestVersion, "v")
			if latest != "" && latest != current {
				updateAvailable = latest
			}
			return
		}

		// Fetch from GitHub
		rel, err := fetchLatestRelease()
		if err != nil {
			return // fail silently
		}

		latest := strings.TrimPrefix(rel.TagName, "v")
		writeCache(&updateCache{
			CheckedAt:     time.Now(),
			LatestVersion: latest,
		})

		current := strings.TrimPrefix(Version, "v")
		if latest != current {
			updateAvailable = latest
		}
	}()
}

func printUpdateNotice() {
	if updateAvailable == "" {
		return
	}
	fmt.Println()
	ui.Warn(fmt.Sprintf("A new version of codag is available: %s → %s", Version, updateAvailable))
	ui.Info("Run `codag upgrade` to update.")
}

func readCache() (*updateCache, error) {
	data, err := os.ReadFile(cacheFilePath())
	if err != nil {
		return nil, err
	}
	var c updateCache
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

func writeCache(c *updateCache) {
	data, err := json.Marshal(c)
	if err != nil {
		return
	}
	os.MkdirAll(config.CodagHome, 0755)
	os.WriteFile(cacheFilePath(), data, 0600)
}
