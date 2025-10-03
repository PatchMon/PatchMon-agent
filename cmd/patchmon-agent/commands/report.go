package commands

import (
	"context"
	"fmt"

	"patchmon-agent/internal/client"
	"patchmon-agent/internal/hardware"
	"patchmon-agent/internal/network"
	"patchmon-agent/internal/packages"
	"patchmon-agent/internal/repositories"
	"patchmon-agent/internal/system"
	"patchmon-agent/internal/version"
	"patchmon-agent/pkg/models"

	"github.com/sirupsen/logrus"
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

	// Load API credentials to send report
	logger.Debug("Loading API credentials")
	if err := cfgManager.LoadCredentials(); err != nil {
		logger.WithError(err).Debug("Failed to load credentials")
		return err
	}

	// Initialise managers
	systemDetector := system.New(logger)
	packageMgr := packages.New(logger)
	repoMgr := repositories.New(logger)
	hardwareMgr := hardware.New(logger)
	networkMgr := network.New(logger)

	// Detect OS
	logger.Info("Detecting operating system...")
	osType, osVersion, err := systemDetector.DetectOS()
	if err != nil {
		return fmt.Errorf("failed to detect OS: %w", err)
	}
	logger.WithFields(logrus.Fields{
		"osType":    osType,
		"osVersion": osVersion,
	}).Info("Detected OS")

	// Get system information
	logger.Info("Collecting system information...")
	hostname, err := systemDetector.GetHostname()
	if err != nil {
		return fmt.Errorf("failed to get hostname: %w", err)
	}

	architecture := systemDetector.GetArchitecture()
	systemInfo := systemDetector.GetSystemInfo()
	ipAddress := systemDetector.GetIPAddress()

	// Get hardware information
	logger.Info("Collecting hardware information...")
	hardwareInfo := hardwareMgr.GetHardwareInfo()

	// Get network information
	logger.Info("Collecting network information...")
	networkInfo := networkMgr.GetNetworkInfo()

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
	logger.WithField("count", len(packageList)).Info("Found packages")
	for _, pkg := range packageList {
		updateMsg := ""
		if pkg.NeedsUpdate {
			updateMsg = "update available"
		} else {
			updateMsg = "latest"
		}
		logger.WithFields(logrus.Fields{
			"name":    pkg.Name,
			"version": pkg.CurrentVersion,
			"status":  updateMsg,
		}).Debug("Package info")
	}
	logger.WithFields(logrus.Fields{
		"total_updates":    needsUpdateCount,
		"security_updates": securityUpdateCount,
	}).Debug("Package summary")

	// Get repository information
	logger.Info("Collecting repository information...")
	repoList, err := repoMgr.GetRepositories(osType)
	if err != nil {
		logger.WithError(err).Warn("Failed to get repositories")
		repoList = []models.Repository{}
	}
	logger.WithField("count", len(repoList)).Info("Found repositories")
	for _, repo := range repoList {
		logger.WithFields(logrus.Fields{
			"name":    repo.Name,
			"type":    repo.RepoType,
			"url":     repo.URL,
			"enabled": repo.IsEnabled,
		}).Debug("Repository info")
	}

	// Create payload
	payload := &models.ReportPayload{
		Packages:          packageList,
		Repositories:      repoList,
		OSType:            osType,
		OSVersion:         osVersion,
		Hostname:          hostname,
		IP:                ipAddress,
		Architecture:      architecture,
		AgentVersion:      version.Version,
		KernelVersion:     systemInfo.KernelVersion,
		SELinuxStatus:     systemInfo.SELinuxStatus,
		SystemUptime:      systemInfo.SystemUptime,
		LoadAverage:       systemInfo.LoadAverage,
		CPUModel:          hardwareInfo.CPUModel,
		CPUCores:          hardwareInfo.CPUCores,
		RAMInstalled:      hardwareInfo.RAMInstalled,
		SwapSize:          hardwareInfo.SwapSize,
		DiskDetails:       hardwareInfo.DiskDetails,
		GatewayIP:         networkInfo.GatewayIP,
		DNSServers:        networkInfo.DNSServers,
		NetworkInterfaces: networkInfo.NetworkInterfaces,
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
	logger.WithField("count", response.PackagesProcessed).Info("Processed packages")

	// Handle agent auto-update
	if response.AutoUpdate != nil && response.AutoUpdate.ShouldUpdate {
		logger.WithFields(logrus.Fields{
			"current": response.AutoUpdate.CurrentVersion,
			"latest":  response.AutoUpdate.LatestVersion,
			"message": response.AutoUpdate.Message,
		}).Info("PatchMon agent update detected")

		logger.Info("Automatically updating PatchMon agent to latest version...")
		if err := updateAgent(); err != nil {
			logger.WithError(err).Warn("PatchMon agent update failed, but data was sent successfully")
		} else {
			logger.Info("PatchMon agent update completed successfully")
		}
	}

	logger.Debug("Report process completed")
	return nil
}
