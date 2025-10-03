package commands

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"patchmon-agent/internal/client"
	"patchmon-agent/internal/crontab"
	"patchmon-agent/internal/version"

	"github.com/spf13/cobra"
)

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
	// Load credentials
	if err := cfgManager.LoadCredentials(); err != nil {
		return err
	}

	logger.Info("Checking for agent updates...")

	// Create client and check version
	httpClient := client.New(cfgManager, logger)
	ctx := context.Background()
	response, err := httpClient.CheckVersion(ctx)
	if err != nil {
		return fmt.Errorf("failed to check for updates: %w", err)
	}

	if response.CurrentVersion != "" && response.CurrentVersion != version.Version {
		logger.Warn("Agent update available!")
		fmt.Printf("  Current version: %s\n", version.Version)
		fmt.Printf("  Latest version: %s\n", response.CurrentVersion)
		if response.ReleaseNotes != "" {
			fmt.Printf("  Release notes: %s\n", response.ReleaseNotes)
		}
		if response.DownloadURL != "" {
			fmt.Printf("  Download URL: %s\n", response.DownloadURL)
		}
		fmt.Printf("\nTo update, run: patchmon-agent update-agent\n")
	} else {
		logger.WithField("version", version.Version).Info("Agent is up to date")
	}

	return nil
}

func updateAgent() error {
	// Load credentials
	if err := cfgManager.LoadCredentials(); err != nil {
		return err
	}

	logger.Info("Updating agent script...")

	// Get current executable path
	executablePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// Create client and get version info
	httpClient := client.New(cfgManager, logger)
	ctx := context.Background()
	versionResponse, err := httpClient.CheckVersion(ctx)
	if err != nil {
		return fmt.Errorf("failed to get update information: %w", err)
	}

	downloadURL := versionResponse.DownloadURL
	logger.Info("Downloading latest agent from server...")

	// Download new version
	newAgentData, err := httpClient.DownloadUpdate(ctx, downloadURL)
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
	testCmd := exec.Command(tempPath, "--version")
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

	logger.Info("Agent updated successfully")
	if versionResponse.CurrentVersion != "" {
		logger.WithField("version", versionResponse.CurrentVersion).Info("Updated to version")
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

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	return os.WriteFile(dst, data, 0755)
}

// updateCrontabCmd represents the update-crontab command
var updateCrontabCmd = &cobra.Command{
	Use:   "update-crontab",
	Short: "Update crontab with current policy",
	Long:  "Update the crontab entry with the current update interval policy from the server.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := checkRoot(); err != nil {
			return err
		}

		return updateCrontabFromServer()
	},
}

func updateCrontabFromServer() error {
	// Load credentials
	if err := cfgManager.LoadCredentials(); err != nil {
		return err
	}

	logger.Info("Updating crontab with current policy...")

	// Create client and get update interval
	httpClient := client.New(cfgManager, logger)
	ctx := context.Background()
	response, err := httpClient.GetUpdateInterval(ctx)
	if err != nil {
		return fmt.Errorf("failed to get update interval policy: %w", err)
	}

	updateInterval := response.UpdateInterval
	if updateInterval <= 0 {
		return fmt.Errorf("invalid update interval: %d", updateInterval)
	}

	// Get current executable path
	executablePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// Create crontab manager and update schedule
	cronManager := crontab.New(logger)
	return cronManager.UpdateSchedule(updateInterval, executablePath)
}
