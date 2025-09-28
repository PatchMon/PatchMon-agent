package commands

import (
	"context"
	"fmt"

	"patchmon-agent/internal/client"
	"patchmon-agent/internal/packages"
	"patchmon-agent/internal/repositories"
	"patchmon-agent/internal/system"
	"patchmon-agent/internal/version"
	"patchmon-agent/pkg/models"

	"github.com/spf13/cobra"
)

// updateCmd represents the update command
var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Send package update information to server",
	Long:  "Collect and send package and system information to the PatchMon server.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := checkRoot(); err != nil {
			return err
		}

		return sendUpdate()
	},
}

func sendUpdate() error {
	// Load credentials
	if err := cfgManager.LoadCredentials(); err != nil {
		return err
	}

	// Initialise managers
	systemDetector := system.New(logger)
	packageMgr := packages.New(logger)
	repoMgr := repositories.New(logger)

	// Detect OS
	logger.Info("Detecting operating system...")
	osType, osVersion, err := systemDetector.DetectOS()
	if err != nil {
		return fmt.Errorf("failed to detect OS: %w", err)
	}
	logger.Infof("Detected OS: %s %s", osType, osVersion)

	// Get system information
	logger.Info("Collecting system information...")
	hostname, err := systemDetector.GetHostname()
	if err != nil {
		return fmt.Errorf("failed to get hostname: %w", err)
	}

	architecture := systemDetector.GetArchitecture()
	systemInfo := systemDetector.GetSystemInfo()

	// Get package information
	logger.Info("Collecting package information...")
	packageList, err := packageMgr.GetPackages(osType)
	if err != nil {
		return fmt.Errorf("failed to get packages: %w", err)
	}
	logger.Infof("Found %d packages", len(packageList))

	// Get repository information
	logger.Info("Collecting repository information...")
	repoList, err := repoMgr.GetRepositories(osType)
	if err != nil {
		logger.Warnf("Failed to get repositories: %v", err)
		repoList = []models.Repository{}
	}
	logger.Infof("Found %d repositories", len(repoList))

	// Create payload
	payload := &models.UpdatePayload{
		Packages:      packageList,
		Repositories:  repoList,
		OSType:        osType,
		OSVersion:     osVersion,
		Hostname:      hostname,
		Architecture:  architecture,
		AgentVersion:  version.Version,
		KernelVersion: systemInfo.KernelVersion,
		SELinuxStatus: systemInfo.SELinuxStatus,
	}

	// Send update
	logger.Info("Sending update to PatchMon server...")
	httpClient := client.New(cfgManager, logger)
	ctx := context.Background()
	response, err := httpClient.SendUpdate(ctx, payload)
	if err != nil {
		return fmt.Errorf("failed to send update: %w", err)
	}

	logger.Info("Update sent successfully")
	logger.Infof("Processed %d packages", response.PackagesProcessed)

	// Handle auto-update
	if response.AutoUpdate != nil && response.AutoUpdate.ShouldUpdate {
		logger.Infof("PatchMon agent update detected: %s", response.AutoUpdate.Message)
		logger.Infof("Current version: %s, Latest version: %s", response.AutoUpdate.CurrentVersion, response.AutoUpdate.LatestVersion)

		logger.Info("Automatically updating PatchMon agent to latest version...")
		if err := updateAgent(); err != nil {
			logger.Warnf("PatchMon agent update failed, but data was sent successfully: %v", err)
		} else {
			logger.Info("PatchMon agent update completed successfully")
		}
	}

	// Handle crontab update
	if response.CrontabUpdate != nil && response.CrontabUpdate.ShouldUpdate {
		if response.CrontabUpdate.Message != "" {
			logger.Info(response.CrontabUpdate.Message)
		}

		if response.CrontabUpdate.Command == "update-crontab" {
			logger.Info("Automatically updating crontab with new interval...")
			if err := updateCrontabFromServer(); err != nil {
				logger.Warnf("Crontab update failed, but data was sent successfully: %v", err)
			} else {
				logger.Info("Crontab updated successfully")
			}
		}
	}

	return nil
}
