package repositories

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	"patchmon-agent/pkg/models"

	"github.com/sirupsen/logrus"
)

// DNFManager handles dnf/yum repository information collection
type DNFManager struct {
	logger *logrus.Logger
}

// NewDNFManager creates a new DNF repository manager
func NewDNFManager(logger *logrus.Logger) *DNFManager {
	return &DNFManager{
		logger: logger,
	}
}

// GetRepositories gets dnf/yum repository information
func (d *DNFManager) GetRepositories() []models.Repository {
	var repositories []models.Repository

	repoFiles, err := d.findRepoFiles()
	if err != nil {
		d.logger.WithError(err).Error("Failed to find repository files")
		return repositories
	}

	for _, file := range repoFiles {
		repos, err := d.parseRepoFile(file)
		if err != nil {
			d.logger.WithError(err).WithField("file", file).Error("Failed to parse repository file")
			continue
		}
		repositories = append(repositories, repos...)
	}

	return repositories
}

// findRepoFiles finds all .repo files in common locations
func (d *DNFManager) findRepoFiles() ([]string, error) {
	var repoFiles []string
	searchPaths := []string{
		"/etc/yum.repos.d",
		"/etc/dnf/repos.d",
	}

	for _, path := range searchPaths {
		files, err := filepath.Glob(filepath.Join(path, "*.repo"))
		if err != nil {
			d.logger.WithError(err).WithField("path", path).Warn("Failed to search for repo files")
			continue
		}
		repoFiles = append(repoFiles, files...)
	}

	return repoFiles, nil
}

// parseRepoFile parses a .repo file and extracts repository information
func (d *DNFManager) parseRepoFile(filename string) ([]models.Repository, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var repositories []models.Repository
	var currentRepo models.Repository
	var inSection bool

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Check for section header [repo-name]
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			// Save previous repository if it was valid
			if inSection && currentRepo.Name != "" {
				repositories = append(repositories, currentRepo)
			}

			// Start new repository
			currentRepo = models.Repository{
				Name:     strings.Trim(line, "[]"),
				RepoType: "rpm",
			}
			inSection = true
			continue
		}

		if !inSection {
			continue
		}

		// Parse key=value pairs
		if strings.Contains(line, "=") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) != 2 {
				continue
			}

			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])

			switch key {
			case "name":
				// Store description in Distribution field for now
				currentRepo.Distribution = value
			case "baseurl":
				currentRepo.URL = value
			case "mirrorlist":
				if currentRepo.URL == "" {
					currentRepo.URL = value
				}
			case "metalink":
				if currentRepo.URL == "" {
					currentRepo.URL = value
				}
			case "enabled":
				currentRepo.IsEnabled = (value == "1" || strings.ToLower(value) == "true")
			case "gpgcheck":
				// Store GPG check status
				currentRepo.IsSecure = (value == "1" || strings.ToLower(value) == "true")
			}
		}
	}

	// Don't forget the last repository
	if inSection && currentRepo.Name != "" {
		repositories = append(repositories, currentRepo)
	}

	return repositories, scanner.Err()
}
