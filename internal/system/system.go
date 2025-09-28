package system

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/sirupsen/logrus"
	"patchmon-agent/pkg/models"
)

// Detector handles system information detection
type Detector struct {
	logger *logrus.Logger
}

// New creates a new system detector
func New(logger *logrus.Logger) *Detector {
	return &Detector{
		logger: logger,
	}
}

// DetectOS detects the operating system and version
func (d *Detector) DetectOS() (osType, osVersion string, err error) {
	// Try /etc/os-release first (most common)
	if data, err := os.ReadFile("/etc/os-release"); err == nil {
		osInfo := parseOSRelease(string(data))
		osType = strings.ToLower(osInfo["ID"])
		osVersion = osInfo["VERSION_ID"]

		// Map OS variations to their appropriate categories
		switch osType {
		case "pop", "linuxmint", "elementary":
			osType = "ubuntu"
		case "rhel", "rocky", "almalinux", "centos":
			osType = "rhel"
		}

		return osType, osVersion, nil
	}

	// Try /etc/redhat-release for older RHEL systems
	if data, err := os.ReadFile("/etc/redhat-release"); err == nil {
		content := string(data)
		if strings.Contains(content, "CentOS") {
			osType = "centos"
		} else if strings.Contains(content, "Red Hat") {
			osType = "rhel"
		}

		// Extract version using a simple approach
		for field := range strings.FieldsSeq(content) {
			if strings.Contains(field, ".") && len(field) <= 10 {
				// Likely a version string
				osVersion = field
				break
			}
		}

		return osType, osVersion, nil
	}

	return "", "", fmt.Errorf("unable to detect OS version")
}

// GetArchitecture returns the system architecture
func (d *Detector) GetArchitecture() string {
	return runtime.GOARCH
}

// GetHostname returns the system hostname
func (d *Detector) GetHostname() (string, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return "", fmt.Errorf("failed to get hostname: %w", err)
	}
	return hostname, nil
}

// GetSystemInfo gets additional system information
func (d *Detector) GetSystemInfo() *models.SystemInfo {
	info := &models.SystemInfo{
		SELinuxStatus: "disabled", // Default value
	}

	// Get kernel version
	if data, err := os.ReadFile("/proc/version"); err == nil {
		fields := strings.Fields(string(data))
		if len(fields) >= 3 {
			info.KernelVersion = fields[2]
		}
	} else {
		// Fallback to uname -r
		if output, err := exec.Command("uname", "-r").Output(); err == nil {
			info.KernelVersion = strings.TrimSpace(string(output))
		}
	}

	// Get SELinux status
	if output, err := exec.Command("getenforce").Output(); err == nil {
		info.SELinuxStatus = strings.ToLower(strings.TrimSpace(string(output)))
	} else if data, err := os.ReadFile("/etc/selinux/config"); err == nil {
		// Parse config file
		scanner := bufio.NewScanner(strings.NewReader(string(data)))
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if value, found := strings.CutPrefix(line, "SELINUX="); found {
				info.SELinuxStatus = strings.ToLower(strings.Trim(value, "\"'"))
				break
			}
		}
	}

	return info
}

// parseOSRelease parses /etc/os-release file content
func parseOSRelease(content string) map[string]string {
	result := make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(content))

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Remove quotes from value
		if len(value) >= 2 && ((value[0] == '"' && value[len(value)-1] == '"') ||
			(value[0] == '\'' && value[len(value)-1] == '\'')) {
			value = value[1 : len(value)-1]
		}

		result[key] = value
	}

	return result
}
