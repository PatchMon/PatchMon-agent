package system

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/sirupsen/logrus"

	"patchmon-agent/internal/constants"
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
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	info, err := host.InfoWithContext(ctx)
	if err != nil {
		d.logger.WithError(err).Warn("Failed to get host info")
		return "", "", err
	}

	osType = info.Platform
	osVersion = info.PlatformVersion

	// Map OS variations to their appropriate categories
	switch osType {
	case constants.OSTypePop, constants.OSTypeMint, "elementary":
		osType = constants.OSTypeUbuntu
	case constants.OSTypeRHEL, constants.OSTypeRocky, constants.OSTypeAlma, constants.OSTypeCentOS:
		osType = constants.OSTypeRHEL
	}

	return osType, osVersion, nil
}

// GetSystemInfo gets additional system information
func (d *Detector) GetSystemInfo() models.SystemInfo {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	d.logger.Debug("Beginning system information collection")

	info := models.SystemInfo{
		KernelVersion: d.GetKernelVersion(ctx),
		SELinuxStatus: d.getSELinuxStatus(),
		SystemUptime:  d.getSystemUptime(ctx),
		LoadAverage:   d.getLoadAverage(ctx),
	}

	d.logger.WithFields(logrus.Fields{
		"kernel":  info.KernelVersion,
		"selinux": info.SELinuxStatus,
		"uptime":  info.SystemUptime,
	}).Debug("Collected kernel, SELinux, and uptime information")

	return info
}

// GetArchitecture returns the system architecture
func (d *Detector) GetArchitecture() string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	info, err := host.InfoWithContext(ctx)
	if err != nil {
		d.logger.WithError(err).Warn("Failed to get architecture")
		return constants.ArchUnknown
	}

	return info.KernelArch
}

// GetHostname returns the system hostname
func (d *Detector) GetHostname() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	info, err := host.InfoWithContext(ctx)
	if err != nil {
		d.logger.WithError(err).Warn("Failed to get hostname")
		// Fallback to os.Hostname
		return os.Hostname()
	}

	return info.Hostname, nil
}

// GetIPAddress gets the primary IP address using network interfaces
func (d *Detector) GetIPAddress() string {
	interfaces, err := net.Interfaces()
	if err != nil {
		d.logger.WithError(err).Warn("Failed to get network interfaces")
		return ""
	}

	for _, iface := range interfaces {
		// Skip loopback and down interfaces
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok {
				if ipnet.IP.To4() != nil && !ipnet.IP.IsLoopback() {
					return ipnet.IP.String()
				}
			}
		}
	}

	return ""
}

// GetKernelVersion gets the kernel version
func (d *Detector) GetKernelVersion(ctx context.Context) string {
	info, err := host.InfoWithContext(ctx)
	if err != nil {
		d.logger.WithError(err).Warn("Failed to get kernel version")
		return constants.ErrUnknownValue
	}

	return info.KernelVersion
}

// getSELinuxStatus gets SELinux status using file reading
func (d *Detector) getSELinuxStatus() string {
	// Try getenforce command first
	if cmd := exec.Command("getenforce"); cmd != nil {
		if output, err := cmd.Output(); err == nil {
			status := strings.ToLower(strings.TrimSpace(string(output)))
			// Map "enforcing" to "enabled" for server validation
			if status == constants.SELinuxEnforcing {
				return constants.SELinuxEnabled
			}
			if status == constants.SELinuxPermissive {
				return constants.SELinuxPermissive
			}
			return status
		}
	}

	// Fallback to reading config file
	if data, err := os.ReadFile("/etc/selinux/config"); err == nil {
		scanner := bufio.NewScanner(strings.NewReader(string(data)))
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if value, found := strings.CutPrefix(line, "SELINUX="); found {
				status := strings.ToLower(strings.Trim(value, "\"'"))
				// Map "enforcing" to "enabled" for server validation
				if status == constants.SELinuxEnforcing {
					return constants.SELinuxEnabled
				}
				if status == constants.SELinuxPermissive {
					return constants.SELinuxPermissive
				}
				return status
			}
		}
	}

	return constants.SELinuxDisabled
}

// getSystemUptime gets system uptime
func (d *Detector) getSystemUptime(ctx context.Context) string {
	info, err := host.InfoWithContext(ctx)
	if err != nil {
		d.logger.WithError(err).Warn("Failed to get uptime")
		return "Unknown"
	}

	uptime := time.Duration(info.Uptime) * time.Second

	days := int(uptime.Hours() / 24)
	hours := int(uptime.Hours()) % 24
	minutes := int(uptime.Minutes()) % 60

	if days > 0 {
		return fmt.Sprintf("%d days, %d hours, %d minutes", days, hours, minutes)
	} else if hours > 0 {
		return fmt.Sprintf("%d hours, %d minutes", hours, minutes)
	} else {
		return fmt.Sprintf("%d minutes", minutes)
	}
}

// getLoadAverage gets system load average
func (d *Detector) getLoadAverage(ctx context.Context) []float64 {
	loadAvg, err := load.AvgWithContext(ctx)
	if err != nil {
		d.logger.WithError(err).Warn("Failed to get load average")
		return []float64{0, 0, 0}
	}

	return []float64{loadAvg.Load1, loadAvg.Load5, loadAvg.Load15}
}
