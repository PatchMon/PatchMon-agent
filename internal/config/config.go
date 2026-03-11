package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"

	"patchmon-agent/pkg/models"

	"github.com/spf13/viper"
)

var (
	DefaultAPIVersion      = "v1"
	DefaultConfigFile      = getDefaultConfigFile()
	DefaultCredentialsFile = getDefaultCredentialsFile()
	DefaultLogFile         = getDefaultLogFile()
	DefaultLogLevel        = "info"
	CronFilePath           = getDefaultCronFile()
)

// getDefaultConfigFile returns the default config file path based on OS
func getDefaultConfigFile() string {
	if runtime.GOOS == "windows" {
		programData := os.Getenv("ProgramData")
		if programData == "" {
			// Fallback to C:\ProgramData if environment variable is not set
			programData = "C:\\ProgramData"
		}
		return filepath.Join(programData, "PatchMon", "config.yml")
	}
	return "/etc/patchmon/config.yml"
}

// getDefaultCredentialsFile returns the default credentials file path based on OS
func getDefaultCredentialsFile() string {
	if runtime.GOOS == "windows" {
		programData := os.Getenv("ProgramData")
		if programData == "" {
			// Fallback to C:\ProgramData if environment variable is not set
			programData = "C:\\ProgramData"
		}
		return filepath.Join(programData, "PatchMon", "credentials.yml")
	}
	return "/etc/patchmon/credentials.yml"
}

// getDefaultLogFile returns the default log file path based on OS
func getDefaultLogFile() string {
	if runtime.GOOS == "windows" {
		programData := os.Getenv("ProgramData")
		if programData == "" {
			// Fallback to C:\ProgramData if environment variable is not set
			programData = "C:\\ProgramData"
		}
		return filepath.Join(programData, "PatchMon", "patchmon-agent.log")
	}
	return "/var/log/patchmon-agent.log"
}

// getDefaultCronFile returns the default cron file path (Linux only)
func getDefaultCronFile() string {
	if runtime.GOOS == "windows" {
		// Windows doesn't use cron files - this shouldn't be used on Windows
		return ""
	}
	return "/etc/cron.d/patchmon-agent"
}

// AvailableIntegrations lists all integrations that can be enabled/disabled
// Add new integrations here as they are implemented
var AvailableIntegrations = []string{
	"docker",
	// Future: "proxmox", "kubernetes", etc.
}

// Manager handles configuration management
type Manager struct {
	config      *models.Config
	credentials *models.Credentials
	configFile  string
}

// New creates a new configuration manager
func New() *Manager {
	return &Manager{
		config: &models.Config{
			PatchmonServer:  "", // No default server - user must provide
			APIVersion:      DefaultAPIVersion,
			CredentialsFile: DefaultCredentialsFile,
			LogFile:         DefaultLogFile,
			LogLevel:        DefaultLogLevel,
			UpdateInterval:  60, // Default to 60 minutes
			Integrations:    make(map[string]bool),
		},
		configFile: DefaultConfigFile,
	}
}

// SetConfigFile sets the path to the config file (called from CLI flag)
func (m *Manager) SetConfigFile(path string) {
	m.configFile = path
}

// GetConfigFile returns the path to the config file
func (m *Manager) GetConfigFile() string {
	return m.configFile
}

// GetConfig returns the current configuration
func (m *Manager) GetConfig() *models.Config {
	return m.config
}

// GetCredentials returns the current credentials
func (m *Manager) GetCredentials() *models.Credentials {
	return m.credentials
}

// LoadConfig loads configuration from file
func (m *Manager) LoadConfig() error {
	// Check if config file exists
	if _, err := os.Stat(m.configFile); errors.Is(err, fs.ErrNotExist) {
		// Use defaults if config file doesn't exist
		return nil
	}

	viper.SetConfigFile(m.configFile)
	viper.SetConfigType("yaml")

	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("error reading config file: %w", err)
	}

	if err := viper.Unmarshal(m.config); err != nil {
		return fmt.Errorf("error unmarshaling config: %w", err)
	}

	// Handle backward compatibility: set defaults for fields that may not exist in older configs
	// If UpdateInterval is 0 or not set, use default of 60 minutes
	if m.config.UpdateInterval <= 0 {
		m.config.UpdateInterval = 60
	}

	// If Integrations map is nil (not set in old configs), initialize it
	if m.config.Integrations == nil {
		m.config.Integrations = make(map[string]bool)
	}

	// Ensure all available integrations are present in the map with default value (false)
	// This ensures config.yml always shows all integrations, even if they're disabled
	for _, integrationName := range AvailableIntegrations {
		if _, exists := m.config.Integrations[integrationName]; !exists {
			m.config.Integrations[integrationName] = false
		}
	}

	// ReportOffset can be 0 - it will be recalculated if missing
	// No need to set a default here as it's calculated dynamically

	return nil
}

// LoadCredentials loads API credentials from file
func (m *Manager) LoadCredentials() error {
	if _, err := os.Stat(m.config.CredentialsFile); errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("credentials file not found at %s", m.config.CredentialsFile)
	}

	viper.New()
	credViper := viper.New()
	credViper.SetConfigFile(m.config.CredentialsFile)
	credViper.SetConfigType("yaml")

	if err := credViper.ReadInConfig(); err != nil {
		return fmt.Errorf("error reading credentials file: %w", err)
	}

	m.credentials = &models.Credentials{}
	if err := credViper.Unmarshal(m.credentials); err != nil {
		return fmt.Errorf("error unmarshaling credentials: %w", err)
	}

	if m.credentials.APIID == "" || m.credentials.APIKey == "" {
		return fmt.Errorf("api_id and api_key must be configured in %s", m.config.CredentialsFile)
	}

	return nil
}

// SaveCredentials saves API credentials to file
func (m *Manager) SaveCredentials(apiID, apiKey string) error {
	if err := m.setupDirectories(); err != nil {
		return err
	}

	m.credentials = &models.Credentials{
		APIID:  apiID,
		APIKey: apiKey,
	}

	credViper := viper.New()
	credViper.Set("api_id", m.credentials.APIID)
	credViper.Set("api_key", m.credentials.APIKey)

	if err := credViper.WriteConfigAs(m.config.CredentialsFile); err != nil {
		return fmt.Errorf("error writing credentials file: %w", err)
	}

	// Set restrictive permissions (Unix-style file permissions, ignored on Windows)
	if runtime.GOOS != "windows" {
		if err := os.Chmod(m.config.CredentialsFile, 0600); err != nil {
			return fmt.Errorf("error setting credentials file permissions: %w", err)
		}
	}
	// On Windows, file permissions are managed via ACLs, not chmod

	return nil
}

// SaveConfig saves configuration to file
func (m *Manager) SaveConfig() error {
	if err := m.setupDirectories(); err != nil {
		return err
	}

	configViper := viper.New()
	configViper.Set("patchmon_server", m.config.PatchmonServer)
	configViper.Set("api_version", m.config.APIVersion)
	configViper.Set("credentials_file", m.config.CredentialsFile)
	configViper.Set("log_file", m.config.LogFile)
	configViper.Set("log_level", m.config.LogLevel)
	configViper.Set("skip_ssl_verify", m.config.SkipSSLVerify)
	configViper.Set("update_interval", m.config.UpdateInterval)
	configViper.Set("report_offset", m.config.ReportOffset)

	// Always save integrations map with all available integrations
	// This ensures config.yml always shows all integrations with their current state
	// Ensure all available integrations are present before saving
	if m.config.Integrations == nil {
		m.config.Integrations = make(map[string]bool)
	}
	for _, integrationName := range AvailableIntegrations {
		if _, exists := m.config.Integrations[integrationName]; !exists {
			m.config.Integrations[integrationName] = false
		}
	}
	configViper.Set("integrations", m.config.Integrations)

	if err := configViper.WriteConfigAs(m.configFile); err != nil {
		return fmt.Errorf("error writing config file: %w", err)
	}

	return nil
}

// SetUpdateInterval sets the update interval and saves it to config file
func (m *Manager) SetUpdateInterval(interval int) error {
	if interval <= 0 {
		return fmt.Errorf("invalid update interval: %d (must be > 0)", interval)
	}
	m.config.UpdateInterval = interval
	return m.SaveConfig()
}

// SetReportOffset sets the report offset (in seconds) and saves it to config file
func (m *Manager) SetReportOffset(offsetSeconds int) error {
	if offsetSeconds < 0 {
		return fmt.Errorf("invalid report offset: %d (must be >= 0)", offsetSeconds)
	}
	m.config.ReportOffset = offsetSeconds
	return m.SaveConfig()
}

// IsIntegrationEnabled checks if an integration is enabled
// Returns false if not specified (default behavior - integrations are disabled by default)
func (m *Manager) IsIntegrationEnabled(name string) bool {
	if m.config.Integrations == nil {
		return false
	}
	enabled, exists := m.config.Integrations[name]
	return exists && enabled
}

// SetIntegrationEnabled sets the enabled status for an integration
func (m *Manager) SetIntegrationEnabled(name string, enabled bool) error {
	if m.config.Integrations == nil {
		m.config.Integrations = make(map[string]bool)
	}
	m.config.Integrations[name] = enabled
	return m.SaveConfig()
}

// setupDirectories creates necessary directories
func (m *Manager) setupDirectories() error {
	dirs := []string{
		filepath.Dir(m.configFile),
		filepath.Dir(m.config.CredentialsFile),
		filepath.Dir(m.config.LogFile),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("error creating directory %s: %w", dir, err)
		}
	}

	return nil
}
