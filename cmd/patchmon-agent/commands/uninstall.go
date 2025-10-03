package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"patchmon-agent/internal/crontab"

	"github.com/spf13/cobra"
)

// uninstallCmd represents the uninstall command
var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Uninstall the PatchMon agent",
	Long: `Completely remove the PatchMon agent from the system.

This command requires root privileges and will prompt for confirmation.

Examples:
  patchmon-agent uninstall                   # Basic uninstall (keeps config/logs)
  patchmon-agent uninstall --remove-config   # Remove config and credentials too
  patchmon-agent uninstall --remove-logs     # Remove log files too
  patchmon-agent uninstall -a                # Remove everything
  patchmon-agent uninstall -af               # Remove everything without confirmation`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := checkRoot(); err != nil {
			return err
		}

		// Get flags
		removeConfig, _ := cmd.Flags().GetBool("remove-config")
		removeLogs, _ := cmd.Flags().GetBool("remove-logs")
		removeAll, _ := cmd.Flags().GetBool("remove-all")
		force, _ := cmd.Flags().GetBool("force")

		// If remove-all is set, enable both config and logs removal
		if removeAll {
			removeConfig = true
			removeLogs = true
		}

		return performUninstall(removeConfig, removeLogs, force)
	},
}

func init() {
	uninstallCmd.Flags().Bool("remove-config", false, "Remove configuration and credentials files")
	uninstallCmd.Flags().Bool("remove-logs", false, "Remove log files")
	uninstallCmd.Flags().BoolP("remove-all", "a", false, "Remove all files (config, credentials, and logs)")
	uninstallCmd.Flags().BoolP("force", "f", false, "Skip confirmation prompts")
}

func performUninstall(removeConfig, removeLogs, force bool) error {
	cfg := cfgManager.GetConfig()

	// Get current executable path
	executablePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// Resolve symlinks to get the actual binary path
	resolvedPath, err := filepath.EvalSymlinks(executablePath)
	if err != nil {
		logger.WithError(err).WithField("path", executablePath).Warn("Could not resolve symlinks")
		resolvedPath = executablePath
	}

	logger.Info("PatchMon Agent Uninstall")
	logger.Info("========================")

	// Show what will be removed
	fmt.Printf("The following items will be removed:\n")
	fmt.Printf("  - Agent binary: %s\n", resolvedPath)

	// Check for common installation locations
	commonPaths := []string{
		"/usr/local/bin/patchmon-agent",
		"/usr/bin/patchmon-agent",
		"/opt/patchmon/patchmon-agent",
	}

	var foundPaths []string
	for _, path := range commonPaths {
		if path != resolvedPath {
			if _, err := os.Stat(path); err == nil {
				foundPaths = append(foundPaths, path)
			}
		}
	}

	if len(foundPaths) > 0 {
		fmt.Printf("  - Additional binaries found:\n")
		for _, path := range foundPaths {
			fmt.Printf("    - %s\n", path)
		}
	}

	// Check for crontab entry
	cronManager := crontab.New(logger)
	crontabEntries := cronManager.GetEntries()
	if len(crontabEntries) > 0 {
		fmt.Printf("  - Crontab entries (%d)\n", len(crontabEntries))
	}

	// Check for backup files
	backupFiles := findBackupFiles(resolvedPath)
	if len(backupFiles) > 0 {
		fmt.Printf("  - Backup files (%d found)\n", len(backupFiles))
	}

	if removeConfig {
		fmt.Printf("  - Configuration files:\n")
		fmt.Printf("    - %s\n", cfgManager.GetConfigFile())
		fmt.Printf("    - %s\n", cfg.CredentialsFile)
		fmt.Printf("    - %s (directory)\n", filepath.Dir(cfgManager.GetConfigFile()))
	}

	if removeLogs {
		fmt.Printf("  - Log files:\n")
		fmt.Printf("    - %s\n", cfg.LogFile)
	}

	fmt.Printf("\n")

	// Confirmation prompt
	if !force {
		fmt.Printf("Are you sure you want to uninstall PatchMon Agent? [y/N]: ")
		var response string
		fmt.Scanln(&response)
		response = strings.ToLower(strings.TrimSpace(response))
		if response != "y" && response != "yes" {
			logger.Info("Uninstall cancelled")
			return nil
		}
	}

	logger.Info("Starting uninstall process...")

	// Remove crontab entry
	if len(crontabEntries) > 0 {
		logger.Info("Removing crontab entries...")
		if err := cronManager.Remove(); err != nil {
			logger.WithError(err).Warn("Failed to remove crontab entries")
		} else {
			logger.Info("Crontab entries removed successfully")
		}
	}

	// Remove backup files
	if len(backupFiles) > 0 {
		logger.Info("Removing backup files...")
		for _, backup := range backupFiles {
			if err := os.Remove(backup); err != nil {
				logger.WithError(err).WithField("path", backup).Warn("Failed to remove backup file")
			} else {
				logger.WithField("path", backup).Info("Removed backup file")
			}
		}
	}

	// Remove additional binaries
	for _, path := range foundPaths {
		logger.WithField("path", path).Info("Removing additional binary")
		if err := os.Remove(path); err != nil {
			logger.WithError(err).WithField("path", path).Warn("Failed to remove binary")
		} else {
			logger.WithField("path", path).Info("Removed binary")
		}
	}

	// Remove configuration files
	if removeConfig {
		logger.Info("Removing configuration files...")

		// Remove credentials file
		if err := os.Remove(cfg.CredentialsFile); err != nil {
			if !os.IsNotExist(err) {
				logger.WithError(err).Warn("Failed to remove credentials file")
			}
		} else {
			logger.WithField("path", cfg.CredentialsFile).Info("Removed credentials file")
		}

		// Remove config file
		configFile := cfgManager.GetConfigFile()
		if err := os.Remove(configFile); err != nil {
			if !os.IsNotExist(err) {
				logger.WithError(err).Warn("Failed to remove config file")
			}
		} else {
			logger.WithField("path", configFile).Info("Removed config file")
		}

		// Try to remove config directory if empty
		configDir := filepath.Dir(configFile)
		if err := os.Remove(configDir); err != nil {
			if !os.IsNotExist(err) {
				logger.WithError(err).Error("Config directory could not be removed")
			}
		} else {
			logger.WithField("path", configDir).Info("Removed config directory")
		}
	}

	// Remove log files
	if removeLogs {
		logger.Info("Removing log files...")
		if err := os.Remove(cfg.LogFile); err != nil {
			if !os.IsNotExist(err) {
				logger.WithError(err).Warn("Failed to remove log file")
			}
		} else {
			logger.WithField("path", cfg.LogFile).Info("Removed log file")
		}
	}

	// Remove main binary (this should be done last since we're running from it)
	logger.WithField("path", resolvedPath).Info("Removing main binary")

	// Create a self-destruct script that will remove the binary after we exit
	selfDestructScript := fmt.Sprintf(`#!/bin/bash
sleep 1
rm -f "%s"
echo "PatchMon Agent uninstall completed successfully"
rm -f "$0"  # Remove this script too
`, resolvedPath)

	scriptPath := "/tmp/patchmon-uninstall.sh"
	if err := os.WriteFile(scriptPath, []byte(selfDestructScript), 0755); err != nil {
		logger.WithError(err).Warn("Failed to create self-destruct script, manual removal required")
		logger.WithField("path", resolvedPath).Info("Please manually remove")
	} else {
		// Execute the self-destruct script in background
		cmd := exec.Command("nohup", "bash", scriptPath)
		if err := cmd.Start(); err != nil {
			logger.WithError(err).Warn("Failed to start self-destruct script")
			logger.WithField("path", resolvedPath).Info("Please manually remove")
		}
	}

	logger.Info("PatchMon Agent uninstall process completed")

	if !removeConfig {
		logger.Info("Configuration files were preserved (--remove-config or --remove-all not set)")
	}

	if !removeLogs {
		logger.Info("Log files were preserved (--remove-logs or --remove-all not set)")
	}

	return nil
}

// findBackupFiles finds backup files created during agent updates
func findBackupFiles(executablePath string) []string {
	var backupFiles []string

	// Look for .backup.* files in the same directory
	dir := filepath.Dir(executablePath)
	basename := filepath.Base(executablePath)

	entries, err := os.ReadDir(dir)
	if err != nil {
		return backupFiles
	}

	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, basename+".backup.") {
			backupFiles = append(backupFiles, filepath.Join(dir, name))
		}
	}

	return backupFiles
}
