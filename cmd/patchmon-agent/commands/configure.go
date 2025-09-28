package commands

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/spf13/cobra"
)

// configureCmd represents the configure command
var configureCmd = &cobra.Command{
	Use:   "configure <API_ID> <API_KEY> <SERVER_URL>",
	Short: "Configure API credentials for this host",
	Long: `Configure API credentials for the PatchMon server.

Example:
  patchmon-agent configure patchmon_1a2b3c4d abcd1234567890abcdef1234567890abcdef1234567890abcdef1234567890 http://patchmon.example.com`,
	Args: cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := checkRoot(); err != nil {
			return err
		}

		apiID := args[0]
		apiKey := args[1]
		serverURL := args[2]

		return configureCreds(apiID, apiKey, serverURL)
	},
}

func configureCreds(apiID, apiKey, serverURL string) error {
	logger.Info("Setting up credentials...")

	// Validate credentials not empty
	if strings.TrimSpace(apiID) == "" || strings.TrimSpace(apiKey) == "" {
		return fmt.Errorf("API ID and API Key must be set")
	}

	// Validate server URL format
	if _, err := url.Parse(serverURL); err != nil {
		return fmt.Errorf("invalid server URL format: %w", err)
	}

	if !strings.HasPrefix(serverURL, "http://") && !strings.HasPrefix(serverURL, "https://") {
		return fmt.Errorf("invalid server URL format. Must start with http:// or https://")
	}

	// Set server URL in config
	cfg := cfgManager.GetConfig()
	cfg.PatchmonServer = serverURL

	// Save config
	if err := cfgManager.SaveConfig(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	// Save credentials
	if err := cfgManager.SaveCredentials(apiID, apiKey); err != nil {
		return fmt.Errorf("failed to save credentials: %w", err)
	}

	logger.Info("Configuration saved successfully")
	logger.Infof("Config saved to: %s", cfgManager.GetConfigFile())
	logger.Infof("Credentials saved to: %s", cfg.CredentialsFile)

	// Test credentials
	logger.Info("Testing connection...")
	return pingServer()
}
