package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"patchmon-agent/pkg/models"

	"github.com/spf13/viper"
)

const (
	DefaultAPIVersion      = "v1"
	DefaultConfigFile      = "/etc/patchmon/config.yml"
	DefaultCredentialsFile = "/etc/patchmon/credentials.yml"
	DefaultLogFile         = "/etc/patchmon/logs/patchmon-agent.log"
	DefaultLogLevel        = "info"
	CronFilePath           = "/etc/cron.d/patchmon-agent"
)

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

	// Set restrictive permissions
	if err := os.Chmod(m.config.CredentialsFile, 0600); err != nil {
		return fmt.Errorf("error setting credentials file permissions: %w", err)
	}

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

	if err := configViper.WriteConfigAs(m.configFile); err != nil {
		return fmt.Errorf("error writing config file: %w", err)
	}

	return nil
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
