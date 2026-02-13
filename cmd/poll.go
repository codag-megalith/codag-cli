package cmd

import (
	"fmt"
	"time"

	"github.com/codag-megalith/codag-cli/internal/api"
	"github.com/codag-megalith/codag-cli/internal/ui"
)

const (
	pollInterval   = 5 * time.Second
	pollTimeout    = 30 * time.Minute
	pollGracePeriod = 2 * time.Minute
)

// pollIndexing polls /api/stats until indexing completes or timeout.
func pollIndexing(client *api.Client, repoID int) (*api.StatsResponse, error) {
	spin := ui.NewSpinner("Waiting for indexing...")
	spin.Start()
	defer spin.Stop()

	lastSignals := 0
	start := time.Now()

	for {
		time.Sleep(pollInterval)

		stats, err := client.GetStats(repoID)
		if err != nil {
			// Transient errors during polling are OK — keep trying
			if time.Since(start) > pollTimeout {
				spin.Stop()
				ui.Warn("Indexing is taking a while. Check back with: codag status")
				return nil, nil
			}
			continue
		}

		if stats.TotalSignals != lastSignals {
			spin.Update(fmt.Sprintf(
				"Waiting for indexing...  PRs: %d | Signals: %d | Danger: %d",
				stats.PRsIndexed, stats.TotalSignals, stats.DangerSignals,
			))
			lastSignals = stats.TotalSignals
		}

		// Break conditions:
		// 1. Indexing done with signals
		// 2. Indexing done with 0 signals after grace period (avoids infinite loop)
		if !stats.Indexing {
			if stats.TotalSignals > 0 || time.Since(start) > pollGracePeriod {
				break
			}
		}

		if time.Since(start) > pollTimeout {
			spin.Stop()
			ui.Warn("Indexing is taking a while. Check back with: codag status")
			return nil, nil
		}
	}

	spin.Stop()

	// Final stats fetch
	stats, err := client.GetStats(repoID)
	if err != nil {
		return nil, err
	}

	ui.Success("Done!")
	ui.Keyval("PRs indexed", fmt.Sprintf("%d", stats.PRsIndexed))
	ui.Keyval("Files w/ signals", fmt.Sprintf("%d", stats.FilesWithSignals))
	ui.Keyval("Total signals", fmt.Sprintf("%d (%d danger)", stats.TotalSignals, stats.DangerSignals))

	return stats, nil
}
