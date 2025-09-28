package commands

import (
	"fmt"

	"patchmon-agent/internal/version"

	"github.com/spf13/cobra"
)

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Show current configuration",
	Long:  "Display the current configuration settings for the PatchMon agent.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return showConfig()
	},
}

func showConfig() error {
	cfg := cfgManager.GetConfig()
	creds := cfgManager.GetCredentials()

	fmt.Printf("Current Configuration:\n")
	if cfg.PatchmonServer != "" {
		fmt.Printf("  Server: %s\n", cfg.PatchmonServer)
	} else {
		fmt.Printf("  Server: Not configured\n")
	}
	fmt.Printf("  Agent Version: %s\n", version.Version)
	fmt.Printf("  Config File: %s\n", cfgManager.GetConfigFile())
	fmt.Printf("  Credentials File: %s\n", cfg.CredentialsFile)
	fmt.Printf("  Log File: %s\n", cfg.LogFile)
	fmt.Printf("  Log Level: %s\n", cfg.LogLevel)

	if creds != nil {
		fmt.Printf("  API ID: %s\n", creds.APIID)
		// Show only first 8 characters of API key for security
		if len(creds.APIKey) >= 8 {
			fmt.Printf("  API Key: %s...\n", creds.APIKey[:8])
		}
	} else {
		fmt.Printf("  Credentials: Not configured\n")
	}

	return nil
}
