package packages

import (
	"fmt"
	"os/exec"
	"strings"

	"patchmon-agent/pkg/models"

	"github.com/sirupsen/logrus"
)

// Manager handles package information collection
type Manager struct {
	logger     *logrus.Logger
	aptManager *APTManager
	dnfManager *DNFManager
}

// New creates a new package manager
func New(logger *logrus.Logger) *Manager {
	aptManager := NewAPTManager(logger)
	dnfManager := NewDNFManager(logger)

	return &Manager{
		logger:     logger,
		aptManager: aptManager,
		dnfManager: dnfManager,
	}
}

// GetPackages gets package information based on OS type
func (m *Manager) GetPackages(osType string) ([]models.Package, error) {
	// Convert OS type to lowercase for comparison
	osTypeLower := strings.ToLower(osType)
	
	// Check for Debian-based distributions (APT)
	if strings.Contains(osTypeLower, "debian") || 
	   strings.Contains(osTypeLower, "ubuntu") || 
	   strings.Contains(osTypeLower, "pop") || 
	   strings.Contains(osTypeLower, "mint") || 
	   strings.Contains(osTypeLower, "elementary") ||
	   strings.Contains(osTypeLower, "kali") ||
	   strings.Contains(osTypeLower, "parrot") {
		return m.aptManager.GetPackages(), nil
	}
	
	// Check for RHEL-based distributions (DNF/YUM)
	if strings.Contains(osTypeLower, "rhel") || 
	   strings.Contains(osTypeLower, "centos") || 
	   strings.Contains(osTypeLower, "rocky") || 
	   strings.Contains(osTypeLower, "alma") || 
	   strings.Contains(osTypeLower, "fedora") ||
	   strings.Contains(osTypeLower, "red hat") {
		return m.dnfManager.GetPackages(), nil
	}
	
	// Fallback: try to detect package manager directly
	if m.detectPackageManager() == "apt" {
		return m.aptManager.GetPackages(), nil
	} else if m.detectPackageManager() == "dnf" || m.detectPackageManager() == "yum" {
		return m.dnfManager.GetPackages(), nil
	}
	
	return nil, fmt.Errorf("unsupported OS type: %s", osType)
}

// detectPackageManager detects which package manager is available on the system
func (m *Manager) detectPackageManager() string {
	// Check for APT first
	if _, err := exec.LookPath("apt"); err == nil {
		return "apt"
	}
	if _, err := exec.LookPath("apt-get"); err == nil {
		return "apt"
	}
	
	// Check for DNF/YUM
	if _, err := exec.LookPath("dnf"); err == nil {
		return "dnf"
	}
	if _, err := exec.LookPath("yum"); err == nil {
		return "yum"
	}
	
	return "unknown"
}

// CombinePackageData combines and deduplicates installed and upgradable package lists
func CombinePackageData(installedPackages map[string]string, upgradablePackages []models.Package) []models.Package {
	var packages []models.Package
	upgradableMap := make(map[string]bool)

	// First, add all upgradable packages
	for _, pkg := range upgradablePackages {
		packages = append(packages, pkg)
		upgradableMap[pkg.Name] = true
	}

	// Then add installed packages that are not upgradable
	for packageName, version := range installedPackages {
		if !upgradableMap[packageName] {
			packages = append(packages, models.Package{
				Name:             packageName,
				CurrentVersion:   version,
				NeedsUpdate:      false,
				IsSecurityUpdate: false,
			})
		}
	}

	return packages
}
