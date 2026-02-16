package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/codag-megalith/codag-cli/internal/config"
	"github.com/codag-megalith/codag-cli/internal/ui"
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
			if latest != "" && isNewer(latest, current) {
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
		if isNewer(latest, current) {
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
	os.MkdirAll(config.CodagHome, 0700)
	os.WriteFile(cacheFilePath(), data, 0600)
}

// isNewer returns true if a > b using semantic versioning comparison.
func isNewer(a, b string) bool {
	aParts := strings.Split(a, ".")
	bParts := strings.Split(b, ".")

	// Pad to same length
	maxLen := len(aParts)
	if len(bParts) > maxLen {
		maxLen = len(bParts)
	}
	for len(aParts) < maxLen {
		aParts = append(aParts, "0")
	}
	for len(bParts) < maxLen {
		bParts = append(bParts, "0")
	}

	// Compare each part
	for i := 0; i < maxLen; i++ {
		aNum, _ := strconv.Atoi(aParts[i])
		bNum, _ := strconv.Atoi(bParts[i])
		if aNum > bNum {
			return true
		}
		if aNum < bNum {
			return false
		}
	}
	return false
}
