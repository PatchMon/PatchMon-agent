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

// reportCmd represents the report command
var reportCmd = &cobra.Command{
	Use:   "report",
	Short: "Report system and package information to server",
	Long:  "Collect and report system, package, and repository information to the PatchMon server.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := checkRoot(); err != nil {
			return err
		}

		return sendReport()
	},
}

func sendReport() error {
	logger.Debug("Starting report process")

	// Load credentials
	if err := cfgManager.LoadCredentials(); err != nil {
		logger.Debugf("Failed to load credentials: %v", err)
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
	logger.Debugf("System info - Hostname: %s, Architecture: %s, Kernel: %s",
		hostname, architecture, systemInfo.KernelVersion)

	// Get package information
	logger.Info("Collecting package information...")
	packageList, err := packageMgr.GetPackages(osType)
	if err != nil {
		return fmt.Errorf("failed to get packages: %w", err)
	}

	// Count packages for debug logging
	needsUpdateCount := 0
	securityUpdateCount := 0
	for _, pkg := range packageList {
		if pkg.NeedsUpdate {
			needsUpdateCount++
		}
		if pkg.IsSecurityUpdate {
			securityUpdateCount++
		}
	}
	logger.Infof("Found %d packages", len(packageList))
	for _, pkg := range packageList {
		updateMsg := ""
		if pkg.NeedsUpdate {
			updateMsg = "update available"
		} else {
			updateMsg = "latest"
		}
		logger.Debugf("Package: %s - %s (%s)",
			pkg.Name, pkg.CurrentVersion, updateMsg)
	}
	logger.Debugf("Package breakdown - Updates available: %d, Security updates: %d",
		needsUpdateCount, securityUpdateCount)

	// Get repository information
	logger.Info("Collecting repository information...")
	repoList, err := repoMgr.GetRepositories(osType)
	if err != nil {
		logger.Warnf("Failed to get repositories: %v", err)
		repoList = []models.Repository{}
	}
	logger.Infof("Found %d repositories", len(repoList))
	for _, repo := range repoList {
		logger.Debugf("Repository: %s, Type: %s, URL: %s, Enabled: %t",
			repo.Name, repo.RepoType, repo.URL, repo.IsEnabled)
	}

	// Create payload
	payload := &models.ReportPayload{
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

	// Send report
	logger.Info("Sending report to PatchMon server...")
	httpClient := client.New(cfgManager, logger)
	ctx := context.Background()
	response, err := httpClient.SendUpdate(ctx, payload)
	if err != nil {
		return fmt.Errorf("failed to send report: %w", err)
	}

	logger.Info("Report sent successfully")
	logger.Infof("Processed %d packages", response.PackagesProcessed)

	// Handle agent auto-update
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

	logger.Debug("Report process completed")
	return nil
}
