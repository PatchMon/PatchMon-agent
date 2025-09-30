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
		updateLogLevel(cmd)
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

	// Initialise configuration manager
	cfgManager = config.New()
	cfgManager.SetConfigFile(configFile)
}

// updateLogLevel sets the logger level based on the flag value
func updateLogLevel(cmd *cobra.Command) {
	// Load configuration first
	if err := cfgManager.LoadConfig(); err != nil {
		logger.Warnf("Failed to load config: %v", err)
	}

	// Check if the log-level flag was explicitly set
	flagLogLevel := logLevel
	if cmd.Flag("log-level").Changed {
		// Flag was explicitly set, use it
		level, err := logrus.ParseLevel(flagLogLevel)
		if err != nil {
			level = logrus.InfoLevel
		}
		logger.SetLevel(level)
		cfgManager.GetConfig().LogLevel = flagLogLevel
	} else {
		// Flag was not set, use config file value if available
		configLogLevel := cfgManager.GetConfig().LogLevel
		if configLogLevel != "" {
			level, err := logrus.ParseLevel(configLogLevel)
			if err != nil {
				level = logrus.InfoLevel
			}
			logger.SetLevel(level)
		} else {
			// No config value either, use default
			logger.SetLevel(logrus.InfoLevel)
			cfgManager.GetConfig().LogLevel = "info"
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
