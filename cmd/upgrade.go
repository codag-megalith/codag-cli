package cmd

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/codag-megalith/codag-cli/internal/ui"
	"github.com/spf13/cobra"
)

const repoOwner = "codag-megalith"
const repoName = "codag-cli"

type ghRelease struct {
	TagName string    `json:"tag_name"`
	Assets  []ghAsset `json:"assets"`
}

type ghAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrade codag to the latest version",
	RunE: func(cmd *cobra.Command, args []string) error {
		force, _ := cmd.Flags().GetBool("force")

		if Version == "dev" && !force {
			ui.Warn("Running a dev build — cannot determine current version.")
			ui.Info("Use --force to upgrade anyway.")
			return nil
		}

		sp := ui.NewSpinner("Checking for updates…")
		sp.Start()

		release, err := fetchLatestRelease()
		sp.Stop()
		if err != nil {
			return fmt.Errorf("failed to check for updates: %w", err)
		}

		latest := strings.TrimPrefix(release.TagName, "v")
		current := strings.TrimPrefix(Version, "v")

		if latest == current && !force {
			ui.Success(fmt.Sprintf("Already up to date (%s)", current))
			return nil
		}

		assetName := expectedAssetName()
		var asset *ghAsset
		for i := range release.Assets {
			if release.Assets[i].Name == assetName {
				asset = &release.Assets[i]
				break
			}
		}
		if asset == nil {
			return fmt.Errorf("no release asset found for %s/%s (%s)", runtime.GOOS, runtime.GOARCH, assetName)
		}

		sp = ui.NewSpinner(fmt.Sprintf("Downloading v%s…", latest))
		sp.Start()

		tmpDir, err := os.MkdirTemp("", "codag-upgrade-*")
		if err != nil {
			sp.Stop()
			return fmt.Errorf("failed to create temp dir: %w", err)
		}
		defer os.RemoveAll(tmpDir)

		archivePath := filepath.Join(tmpDir, assetName)
		if err := downloadFile(asset.BrowserDownloadURL, archivePath); err != nil {
			sp.Stop()
			return fmt.Errorf("download failed: %w", err)
		}

		sp.Update("Extracting…")

		binaryPath, err := extractBinary(archivePath, tmpDir)
		if err != nil {
			sp.Stop()
			return fmt.Errorf("extraction failed: %w", err)
		}

		sp.Update("Installing…")

		execPath, err := os.Executable()
		if err != nil {
			sp.Stop()
			return fmt.Errorf("cannot locate current binary: %w", err)
		}
		execPath, err = filepath.EvalSymlinks(execPath)
		if err != nil {
			sp.Stop()
			return fmt.Errorf("cannot resolve symlink: %w", err)
		}

		if err := replaceBinary(binaryPath, execPath); err != nil {
			sp.Stop()
			return fmt.Errorf("upgrade failed: %w", err)
		}

		sp.Stop()
		ui.Success(fmt.Sprintf("Upgraded codag: %s → %s", current, latest))
		return nil
	},
}

func init() {
	upgradeCmd.Flags().Bool("force", false, "Force upgrade even on dev builds")
}

func fetchLatestRelease() (*ghRelease, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", repoOwner, repoName)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned %d", resp.StatusCode)
	}

	var rel ghRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, err
	}
	return &rel, nil
}

func expectedAssetName() string {
	ext := ".tar.gz"
	if runtime.GOOS == "windows" {
		ext = ".zip"
	}
	return fmt.Sprintf("codag_%s_%s%s", runtime.GOOS, runtime.GOARCH, ext)
}

func downloadFile(url, dest string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned %d", resp.StatusCode)
	}

	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)
	return err
}

func extractBinary(archivePath, destDir string) (string, error) {
	if strings.HasSuffix(archivePath, ".zip") {
		return extractZip(archivePath, destDir)
	}
	return extractTarGz(archivePath, destDir)
}

func extractTarGz(archivePath, destDir string) (string, error) {
	f, err := os.Open(archivePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return "", err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}

		name := filepath.Base(hdr.Name)
		if name == "codag" || name == "codag.exe" {
			outPath := filepath.Join(destDir, name)
			out, err := os.OpenFile(outPath, os.O_CREATE|os.O_WRONLY, 0755)
			if err != nil {
				return "", err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return "", err
			}
			out.Close()
			return outPath, nil
		}
	}
	return "", fmt.Errorf("binary not found in archive")
}

func extractZip(archivePath, destDir string) (string, error) {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return "", err
	}
	defer r.Close()

	for _, f := range r.File {
		name := filepath.Base(f.Name)
		if name == "codag" || name == "codag.exe" {
			rc, err := f.Open()
			if err != nil {
				return "", err
			}
			outPath := filepath.Join(destDir, name)
			out, err := os.OpenFile(outPath, os.O_CREATE|os.O_WRONLY, 0755)
			if err != nil {
				rc.Close()
				return "", err
			}
			if _, err := io.Copy(out, rc); err != nil {
				out.Close()
				rc.Close()
				return "", err
			}
			out.Close()
			rc.Close()
			return outPath, nil
		}
	}
	return "", fmt.Errorf("binary not found in archive")
}

func replaceBinary(newBinary, target string) error {
	// Get permissions from existing binary
	info, err := os.Stat(target)
	if err != nil {
		return err
	}

	// Atomic replace: rename new over old (works on same filesystem)
	// Move new binary next to target first to ensure same filesystem
	staged := target + ".new"
	if err := copyFile(newBinary, staged); err != nil {
		return err
	}
	if err := os.Chmod(staged, info.Mode()); err != nil {
		os.Remove(staged)
		return err
	}
	if err := os.Rename(staged, target); err != nil {
		os.Remove(staged)
		return err
	}
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
