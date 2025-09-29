package commands

import (
	"fmt"
	"os"

	"patchmon-agent/internal/config"
	"patchmon-agent/internal/version"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	cfgManager *config.Manager
	logger     *logrus.Logger
	configFile string
	logLevel   string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "patchmon-agent",
	Short: "PatchMon Agent for package monitoring",
	Long: `PatchMon Agent v` + version.Version + `

A monitoring agent that sends package update information to PatchMon.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		initialiseAgent()
	},
}

// Execute adds all child commands to the root command and sets flags appropriately
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Set default values
	configFile = config.DefaultConfigFile
	logLevel = config.DefaultLogLevel

	// Add global flags
	rootCmd.PersistentFlags().StringVar(&configFile, "config", configFile, "config file path")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", logLevel, "log level (debug, info, warn, error)")

	// Add all subcommands
	rootCmd.AddCommand(updateCmd)
	rootCmd.AddCommand(pingCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(checkVersionCmd)
	rootCmd.AddCommand(updateAgentCmd)
	rootCmd.AddCommand(updateCrontabCmd)
	rootCmd.AddCommand(diagnosticsCmd)
	rootCmd.AddCommand(uninstallCmd)
}

// initialiseAgent initialises the configuration manager and logger
func initialiseAgent() {
	// Initialise logger
	logger = logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{
		DisableTimestamp: false,
		FullTimestamp:    true,
		TimestampFormat:  "2006-01-02T15:04:05",
	})

	// Set log level
	level, err := logrus.ParseLevel(logLevel)
	if err != nil {
		level = logrus.InfoLevel
	}
	logger.SetLevel(level)

	// Initialise configuration manager
	cfgManager = config.New()
	cfgManager.SetConfigFile(configFile)
	cfgManager.GetConfig().LogLevel = logLevel

	// Load configuration
	if err := cfgManager.LoadConfig(); err != nil {
		logger.Warnf("Failed to load config: %v", err)
	}

	// Update log level from config if it was loaded
	if cfgManager.GetConfig().LogLevel != "" {
		if level, err := logrus.ParseLevel(cfgManager.GetConfig().LogLevel); err == nil {
			logger.SetLevel(level)
		}
	}
}

// checkRoot ensures the command is run as root
func checkRoot() error {
	if os.Geteuid() != 0 {
		return fmt.Errorf("this command must be run as root")
	}
	return nil
}
