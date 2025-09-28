package commands

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"patchmon-agent/internal/client"
)

// pingCmd represents the ping command
var pingCmd = &cobra.Command{
	Use:   "ping",
	Short: "Test connectivity and credentials",
	Long:  "Test connectivity to the PatchMon server and validate API credentials.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := checkRoot(); err != nil {
			return err
		}

		return pingServer()
	},
}

func pingServer() error {
	// Load credentials
	if err := cfgManager.LoadCredentials(); err != nil {
		return fmt.Errorf("failed to load credentials: %w", err)
	}

	// Create client and ping
	httpClient := client.New(cfgManager, logger)
	ctx := context.Background()
	response, err := httpClient.Ping(ctx)
	if err != nil {
		return fmt.Errorf("connectivity test failed: %w", err)
	}

	logger.Info("✅ Connectivity test successful")
	logger.Info("✅ API credentials are valid")
	if response.Hostname != "" {
		logger.Infof("Connected as host: %s", response.Hostname)
	}

	// Check for crontab update
	if response.CrontabUpdate != nil && response.CrontabUpdate.ShouldUpdate {
		if response.CrontabUpdate.Message != "" {
			logger.Info(response.CrontabUpdate.Message)
		}

		if response.CrontabUpdate.Command == "update-crontab" {
			logger.Info("Automatically updating crontab with new interval...")
			if err := updateCrontabFromServer(); err != nil {
				logger.Warnf("Crontab update failed, but ping was successful: %v", err)
			} else {
				logger.Info("Crontab updated successfully")
			}
		}
	}

	return nil
}
