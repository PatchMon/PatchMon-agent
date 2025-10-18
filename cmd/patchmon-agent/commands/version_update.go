package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"patchmon-agent/internal/version"

	"github.com/spf13/cobra"
)

const (
	githubAPIURL  = "https://api.github.com/repos/PatchMon/patchmon-agent/releases/latest"
	githubTimeout = 30 * time.Second
)

// GitHubRelease represents a GitHub release
type GitHubRelease struct {
	TagName string `json:"tag_name"`
	Name    string `json:"name"`
	Body    string `json:"body"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

// checkVersionCmd represents the check-version command
var checkVersionCmd = &cobra.Command{
	Use:   "check-version",
	Short: "Check for agent updates",
	Long:  "Check if there are any updates available for the PatchMon agent.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := checkRoot(); err != nil {
			return err
		}

		return checkVersion()
	},
}

// updateAgentCmd represents the update-agent command
var updateAgentCmd = &cobra.Command{
	Use:   "update-agent",
	Short: "Update agent to latest version",
	Long:  "Download and install the latest version of the PatchMon agent.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := checkRoot(); err != nil {
			return err
		}

		return updateAgent()
	},
}

func checkVersion() error {
	logger.Info("Checking for agent updates...")

	release, err := getLatestRelease()
	if err != nil {
		return fmt.Errorf("failed to check for updates: %w", err)
	}

	latestVersion := strings.TrimPrefix(release.TagName, "v")
	currentVersion := strings.TrimPrefix(version.Version, "v")

	if latestVersion != currentVersion {
		logger.Info("Agent update available!")
		fmt.Printf("  Current version: %s\n", currentVersion)
		fmt.Printf("  Latest version: %s\n", latestVersion)

		fmt.Printf("\nTo update, run: patchmon-agent update-agent\n")
	} else {
		logger.WithField("version", currentVersion).Info("Agent is up to date")
	}

	return nil
}

func updateAgent() error {
	logger.Info("Updating agent...")

	// Get current executable path
	executablePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// Get latest release info
	release, err := getLatestRelease()
	if err != nil {
		return fmt.Errorf("failed to get release information: %w", err)
	}

	latestVersion := strings.TrimPrefix(release.TagName, "v")
	logger.WithField("version", latestVersion).Info("Found latest version")

	// Determine the correct asset name for this platform
	assetName := getAssetName()
	var downloadURL string
	for _, asset := range release.Assets {
		if asset.Name == assetName {
			downloadURL = asset.BrowserDownloadURL
			break
		}
	}

	if downloadURL == "" {
		return fmt.Errorf("no compatible binary found for %s-%s in release %s", runtime.GOOS, runtime.GOARCH, release.TagName)
	}

	logger.WithField("url", downloadURL).Info("Downloading latest agent...")

	// Download new version
	newAgentData, err := downloadBinary(downloadURL)
	if err != nil {
		return fmt.Errorf("failed to download new agent: %w", err)
	}

	// Create backup of current executable
	backupPath := fmt.Sprintf("%s.backup.%s", executablePath, time.Now().Format("20060102_150405"))
	if err := copyFile(executablePath, backupPath); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}
	logger.WithField("path", backupPath).Info("Backup saved")

	// Write new version to temporary file
	tempPath := executablePath + ".new"
	if err := os.WriteFile(tempPath, newAgentData, 0755); err != nil {
		return fmt.Errorf("failed to write new agent: %w", err)
	}

	// Verify the new executable works
	testCmd := exec.Command(tempPath, "check-version")
	if err := testCmd.Run(); err != nil {
		if removeErr := os.Remove(tempPath); removeErr != nil {
			logger.WithError(removeErr).Warn("Failed to remove temporary file after validation failure")
		}
		return fmt.Errorf("new agent executable is invalid: %w", err)
	}

	// Replace current executable
	if err := os.Rename(tempPath, executablePath); err != nil {
		if removeErr := os.Remove(tempPath); removeErr != nil {
			logger.WithError(removeErr).Warn("Failed to remove temporary file after rename failure")
		}
		return fmt.Errorf("failed to replace executable: %w", err)
	}

	logger.WithField("version", latestVersion).Info("Agent updated successfully")

	// Send updated information to PatchMon
	logger.Info("Sending updated information to PatchMon...")
	if err := sendReport(); err != nil {
		logger.WithError(err).Warn("Failed to send updated information to PatchMon (this is not critical)")
	} else {
		logger.Info("Successfully sent updated information to PatchMon")
	}

	return nil
}

// getLatestRelease fetches the latest release from GitHub
func getLatestRelease() (*GitHubRelease, error) {
	ctx, cancel := context.WithTimeout(context.Background(), githubTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", githubAPIURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", fmt.Sprintf("patchmon-agent/%s", version.Version))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			logger.WithError(closeErr).Debug("Failed to close response body")
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("failed to decode release info: %w", err)
	}

	return &release, nil
}

// downloadBinary downloads a binary from the given URL
func downloadBinary(url string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", fmt.Sprintf("patchmon-agent/%s", version.Version))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			logger.WithError(closeErr).Debug("Failed to close response body")
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return data, nil
}

// getAssetName returns the expected asset name for the current platform
func getAssetName() string {
	return fmt.Sprintf("patchmon-agent-%s-%s", runtime.GOOS, runtime.GOARCH)
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	return os.WriteFile(dst, data, 0755)
}

// Removed update-crontab command (cron is no longer used)
