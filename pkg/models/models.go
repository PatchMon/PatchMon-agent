package models

// Package represents a software package
type Package struct {
	Name             string `json:"name"`
	CurrentVersion   string `json:"currentVersion"`
	AvailableVersion string `json:"availableVersion,omitempty"`
	NeedsUpdate      bool   `json:"needsUpdate"`
	IsSecurityUpdate bool   `json:"isSecurityUpdate"`
}

// Repository represents a software repository
type Repository struct {
	Name         string `json:"name"`
	URL          string `json:"url"`
	Distribution string `json:"distribution"`
	Components   string `json:"components"`
	RepoType     string `json:"repoType"`
	IsEnabled    bool   `json:"isEnabled"`
	IsSecure     bool   `json:"isSecure"`
}

// SystemInfo represents system information
type SystemInfo struct {
	KernelVersion string `json:"kernelVersion"`
	SELinuxStatus string `json:"selinuxStatus"`
}

// UpdatePayload represents the data sent to the server
type UpdatePayload struct {
	Packages      []Package    `json:"packages"`
	Repositories  []Repository `json:"repositories"`
	OSType        string       `json:"osType"`
	OSVersion     string       `json:"osVersion"`
	Hostname      string       `json:"hostname"`
	Architecture  string       `json:"architecture"`
	AgentVersion  string       `json:"agentVersion"`
	KernelVersion string       `json:"kernelVersion"`
	SELinuxStatus string       `json:"selinuxStatus"`
}

// PingResponse represents server ping response
type PingResponse struct {
	Message       string             `json:"message"`
	Timestamp     string             `json:"timestamp"`
	FriendlyName  string             `json:"friendlyName"`
	CrontabUpdate *CrontabUpdateInfo `json:"crontabUpdate,omitempty"`
}

// UpdateResponse represents server update response
type UpdateResponse struct {
	Message           string             `json:"message"`
	PackagesProcessed int                `json:"packagesProcessed"`
	UpdatesAvailable  int                `json:"updatesAvailable,omitempty"`
	SecurityUpdates   int                `json:"securityUpdates,omitempty"`
	AutoUpdate        *AutoUpdateInfo    `json:"autoUpdate,omitempty"`
	CrontabUpdate     *CrontabUpdateInfo `json:"crontabUpdate,omitempty"`
}

// AutoUpdateInfo represents agent auto-update information
type AutoUpdateInfo struct {
	ShouldUpdate   bool   `json:"shouldUpdate"`
	LatestVersion  string `json:"latestVersion"`
	CurrentVersion string `json:"currentVersion"`
	Message        string `json:"message"`
}

// CrontabUpdateInfo represents crontab update information
type CrontabUpdateInfo struct {
	ShouldUpdate bool   `json:"shouldUpdate"`
	Message      string `json:"message"`
	Command      string `json:"command"`
}

// VersionResponse represents version check response
type VersionResponse struct {
	CurrentVersion string `json:"currentVersion"`
	DownloadURL    string `json:"downloadUrl"`
	ReleaseNotes   string `json:"releaseNotes"`
}

// UpdateIntervalResponse represents update interval response
type UpdateIntervalResponse struct {
	UpdateInterval int `json:"updateInterval"`
}

// Credentials holds API authentication information
type Credentials struct {
	APIID  string `yaml:"api_id" mapstructure:"api_id"`
	APIKey string `yaml:"api_key" mapstructure:"api_key"`
}

// Config represents agent configuration
type Config struct {
	PatchmonServer  string `yaml:"patchmon_server" mapstructure:"patchmon_server"`
	APIVersion      string `yaml:"api_version" mapstructure:"api_version"`
	CredentialsFile string `yaml:"credentials_file" mapstructure:"credentials_file"`
	LogFile         string `yaml:"log_file" mapstructure:"log_file"`
	LogLevel        string `yaml:"log_level" mapstructure:"log_level"`
}
