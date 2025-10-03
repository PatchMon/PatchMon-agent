package repositories

import (
	"patchmon-agent/internal/constants"
	"patchmon-agent/pkg/models"

	"github.com/sirupsen/logrus"
)

// Manager handles repository information collection
type Manager struct {
	logger     *logrus.Logger
	aptManager *APTManager
	dnfManager *DNFManager
}

// New creates a new repository manager
func New(logger *logrus.Logger) *Manager {
	return &Manager{
		logger:     logger,
		aptManager: NewAPTManager(logger),
		dnfManager: NewDNFManager(logger),
	}
}

// GetRepositories gets repository information based on OS type
func (m *Manager) GetRepositories(osType string) ([]models.Repository, error) {
	switch osType {
	case constants.OSTypeUbuntu, constants.OSTypeDebian:
		return m.aptManager.GetRepositories()
	case constants.OSTypeCentOS, constants.OSTypeRHEL, constants.OSTypeFedora, constants.OSTypeRocky, constants.OSTypeAlma:
		repos := m.dnfManager.GetRepositories()
		return repos, nil
	default:
		// Return empty slice for unsupported OS types
		return []models.Repository{}, nil
	}
}
