package commands

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"patchmon-agent/internal/config"
	"patchmon-agent/internal/version"

	"github.com/spf13/cobra"
)

const (
	serverTimeout = 30 * time.Second
)

type ServerVersionResponse struct {
	Version      string `json:"version"`
	Architecture string `json:"architecture"`
	Size         int64  `json:"size"`
	Hash         string `json:"hash"`
	DownloadURL  string `json:"downloadUrl"`
	BinaryData   []byte `json:"-"` // Binary data (not serialized to JSON)
}

type ServerVersionInfo struct {
	CurrentVersion         string   `json:"currentVersion"`
	LatestVersion          string   `json:"latestVersion"`
	HasUpdate              bool     `json:"hasUpdate"`
	LastChecked            string   `json:"lastChecked"`
	SupportedArchitectures []string `json:"supportedArchitectures"`
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

	versionInfo, err := getServerVersionInfo()
	if err != nil {
		return fmt.Errorf("failed to check for updates: %w", err)
	}

	currentVersion := strings.TrimPrefix(version.Version, "v")
	latestVersion := strings.TrimPrefix(versionInfo.LatestVersion, "v")

	if versionInfo.HasUpdate {
		logger.Info("Agent update available!")
		fmt.Printf("  Current version: %s\n", currentVersion)
		fmt.Printf("  Latest version: %s\n", latestVersion)
		fmt.Printf("\nTo update, run: patchmon-agent update-agent\n")
	} else {
		logger.WithField("version", currentVersion).Info("Agent is up to date")
		fmt.Printf("Agent is up to date (version %s)\n", currentVersion)
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

	// Get latest binary info from server
	binaryInfo, err := getLatestBinaryFromServer()
	if err != nil {
		return fmt.Errorf("failed to get latest binary information: %w", err)
	}

	logger.WithField("version", binaryInfo.Version).Info("Found latest version")

	logger.Info("Using downloaded agent binary...")

	// Use the binary data directly from the server response
	newAgentData := binaryInfo.BinaryData
	if len(newAgentData) == 0 {
		return fmt.Errorf("no binary data received from server")
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

	logger.WithField("version", binaryInfo.Version).Info("Agent updated successfully")

	// Restart the systemd service to pick up the new binary
	logger.Info("Restarting patchmon-agent service...")
	if err := restartService(); err != nil {
		logger.WithError(err).Warn("Failed to restart service (this is not critical)")
	} else {
		logger.Info("Service restarted successfully")
	}

	// Send updated information to PatchMon
	logger.Info("Sending updated information to PatchMon...")
	if err := sendReport(); err != nil {
		logger.WithError(err).Warn("Failed to send updated information to PatchMon (this is not critical)")
	} else {
		logger.Info("Successfully sent updated information to PatchMon")
	}

	return nil
}

// getServerVersionInfo fetches version information from the PatchMon server
func getServerVersionInfo() (*ServerVersionInfo, error) {
	cfgManager := config.New()
	if err := cfgManager.LoadConfig(); err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	cfg := cfgManager.GetConfig()

	// Load credentials for API authentication
	if err := cfgManager.LoadCredentials(); err != nil {
		return nil, fmt.Errorf("failed to load credentials: %w", err)
	}
	credentials := cfgManager.GetCredentials()

	architecture := getArchitecture()
	currentVersion := strings.TrimPrefix(version.Version, "v")
	url := fmt.Sprintf("%s/api/v1/hosts/agent/version?arch=%s&type=go&currentVersion=%s", cfg.PatchmonServer, architecture, currentVersion)

	ctx, cancel := context.WithTimeout(context.Background(), serverTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", fmt.Sprintf("patchmon-agent/%s", version.Version))
	req.Header.Set("X-API-ID", credentials.APIID)
	req.Header.Set("X-API-KEY", credentials.APIKey)

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
		return nil, fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	var versionInfo ServerVersionInfo
	if err := json.NewDecoder(resp.Body).Decode(&versionInfo); err != nil {
		return nil, fmt.Errorf("failed to decode version info: %w", err)
	}

	return &versionInfo, nil
}

// getLatestBinaryFromServer fetches the latest binary information from the PatchMon server
func getLatestBinaryFromServer() (*ServerVersionResponse, error) {
	cfgManager := config.New()
	if err := cfgManager.LoadConfig(); err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	cfg := cfgManager.GetConfig()

	// Load credentials for API authentication
	if err := cfgManager.LoadCredentials(); err != nil {
		return nil, fmt.Errorf("failed to load credentials: %w", err)
	}
	credentials := cfgManager.GetCredentials()

	architecture := getArchitecture()
	url := fmt.Sprintf("%s/api/v1/hosts/agent/download?arch=%s", cfg.PatchmonServer, architecture)

	ctx, cancel := context.WithTimeout(context.Background(), serverTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", fmt.Sprintf("patchmon-agent/%s", version.Version))
	req.Header.Set("X-API-ID", credentials.APIID)
	req.Header.Set("X-API-KEY", credentials.APIKey)

	// Configure HTTP client for insecure SSL if needed
	httpClient := http.DefaultClient
	if cfg.SkipSSLVerify {
		logger.Warn("⚠️  SSL certificate verification is disabled for binary download")
		httpClient = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		}
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			logger.WithError(closeErr).Debug("Failed to close response body")
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	// Read the binary data
	binaryData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read binary data: %w", err)
	}

	// Calculate hash
	hash := fmt.Sprintf("%x", sha256.Sum256(binaryData))

	return &ServerVersionResponse{
		Version:      version.Version, // We'll get the actual version from the server later
		Architecture: architecture,
		Size:         int64(len(binaryData)),
		Hash:         hash,
		DownloadURL:  url,
		BinaryData:   binaryData, // Store the binary data directly
	}, nil
}

// getArchitecture returns the architecture string for the current platform
func getArchitecture() string {
	return runtime.GOARCH
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	return os.WriteFile(dst, data, 0755)
}

// restartService restarts the patchmon-agent systemd service
func restartService() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "systemctl", "restart", "patchmon-agent")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to restart service: %w, output: %s", err, string(output))
	}

	logger.WithField("output", string(output)).Debug("Service restart command completed")
	return nil
}

// Removed update-crontab command (cron is no longer used)
