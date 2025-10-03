package constants

// SELinux status constants
const (
	SELinuxEnabled    = "enabled"
	SELinuxDisabled   = "disabled"
	SELinuxPermissive = "permissive"
	SELinuxEnforcing  = "enforcing" // Will be mapped to enabled for API compatibility
)

// OS type constants
const (
	OSTypeDebian     = "debian"
	OSTypeUbuntu     = "ubuntu"
	OSTypeRHEL       = "rhel"
	OSTypeCentOS     = "centos"
	OSTypeFedora     = "fedora"
	OSTypeRocky      = "rocky"
	OSTypeAlma       = "almalinux"
	OSTypePop        = "pop"
	OSTypeMint       = "linuxmint"
	OSTypeElementary = "elementary"
)

// Architecture constants
const (
	ArchX86_64  = "x86_64"
	ArchAMD64   = "amd64"
	ArchARM64   = "arm64"
	ArchAARCH64 = "aarch64"
	ArchUnknown = "unknown"
)

// Network interface types
const (
	NetTypeEthernet = "ethernet"
	NetTypeWiFi     = "wifi"
	NetTypeBridge   = "bridge"
	NetTypeLoopback = "loopback"
)

// IP address families
const (
	IPFamilyIPv4 = "inet"
	IPFamilyIPv6 = "inet6"
)

// Repository type constants
const (
	RepoTypeDeb    = "deb"
	RepoTypeDebSrc = "deb-src"
	RepoTypeRPM    = "rpm"
)

// Log level constants
const (
	LogLevelDebug = "debug"
	LogLevelInfo  = "info"
	LogLevelWarn  = "warn"
	LogLevelError = "error"
)

// Common error messages
const (
	ErrUnknownValue = "Unknown"
)
