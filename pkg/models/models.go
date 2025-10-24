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
	KernelVersion string    `json:"kernelVersion"`
	SELinuxStatus string    `json:"selinuxStatus"`
	SystemUptime  string    `json:"systemUptime"`
	LoadAverage   []float64 `json:"loadAverage"`
}

// HardwareInfo represents hardware information
type HardwareInfo struct {
	CPUModel     string     `json:"cpuModel"`
	CPUCores     int        `json:"cpuCores"`
	RAMInstalled float64    `json:"ramInstalled"` // GB
	SwapSize     float64    `json:"swapSize"`     // GB
	DiskDetails  []DiskInfo `json:"diskDetails"`
}

// DiskInfo represents disk information
type DiskInfo struct {
	Name       string `json:"name"`
	Size       string `json:"size"`
	MountPoint string `json:"mountpoint"`
}

// NetworkInfo represents network information
type NetworkInfo struct {
	GatewayIP         string             `json:"gatewayIp"`
	DNSServers        []string           `json:"dnsServers"`
	NetworkInterfaces []NetworkInterface `json:"networkInterfaces"`
}

// NetworkInterface represents a network interface
type NetworkInterface struct {
	Name      string           `json:"name"`
	Type      string           `json:"type"`
	Addresses []NetworkAddress `json:"addresses"`
}

// NetworkAddress represents an IP address
type NetworkAddress struct {
	Address string `json:"address"`
	Family  string `json:"family"`
}

// ReportPayload represents the data sent to the server
type ReportPayload struct {
	Packages          []Package          `json:"packages"`
	Repositories      []Repository       `json:"repositories"`
	OSType            string             `json:"osType"`
	OSVersion         string             `json:"osVersion"`
	Hostname          string             `json:"hostname"`
	IP                string             `json:"ip"`
	Architecture      string             `json:"architecture"`
	AgentVersion      string             `json:"agentVersion"`
	MachineID         string             `json:"machineId"`
	KernelVersion     string             `json:"kernelVersion"`
	SELinuxStatus     string             `json:"selinuxStatus"`
	SystemUptime      string             `json:"systemUptime"`
	LoadAverage       []float64          `json:"loadAverage"`
	CPUModel          string             `json:"cpuModel"`
	CPUCores          int                `json:"cpuCores"`
	RAMInstalled      float64            `json:"ramInstalled"`
	SwapSize          float64            `json:"swapSize"`
	DiskDetails       []DiskInfo         `json:"diskDetails"`
	GatewayIP         string             `json:"gatewayIp"`
	DNSServers        []string           `json:"dnsServers"`
	NetworkInterfaces []NetworkInterface `json:"networkInterfaces"`
	ExecutionTime     float64            `json:"executionTime"` // Collection time in seconds
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

// AgentTimestampResponse represents agent timestamp response
type AgentTimestampResponse struct {
	Version   string `json:"version"`
	Timestamp int64  `json:"timestamp"`
	Exists    bool   `json:"exists"`
}

// HostSettingsResponse represents host settings response
type HostSettingsResponse struct {
	AutoUpdate     bool `json:"auto_update"`
	HostAutoUpdate bool `json:"host_auto_update"`
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
	SkipSSLVerify   bool   `yaml:"skip_ssl_verify" mapstructure:"skip_ssl_verify"`
}
