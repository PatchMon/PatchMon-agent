package packages

import (
	"fmt"
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
	switch osType {
	case "ubuntu", "debian":
		return m.aptManager.GetPackages(), nil
	case "centos", "rhel", "fedora":
		return m.dnfManager.GetPackages(), nil
	default:
		return nil, fmt.Errorf("unsupported OS type: %s", osType)
	}
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
