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
		return m.aptManager.GetPackages()
	case "centos", "rhel", "fedora":
		return m.dnfManager.GetPackages()
	default:
		return nil, fmt.Errorf("unsupported OS type: %s", osType)
	}
}
