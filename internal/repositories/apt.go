package repositories

import (
	"bufio"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"patchmon-agent/pkg/models"

	"github.com/sirupsen/logrus"
)

// APTManager handles APT repository information collection
type APTManager struct {
	logger *logrus.Logger
}

// NewAPTManager creates a new APT repository manager
func NewAPTManager(logger *logrus.Logger) *APTManager {
	return &APTManager{
		logger: logger,
	}
}

// GetRepositories gets APT repository information
func (m *APTManager) GetRepositories() ([]models.Repository, error) {
	var repositories []models.Repository

	// Find repository files
	listFiles, err := m.findListFiles()
	if err != nil {
		m.logger.WithError(err).Error("Failed to find list files")
		return repositories, err
	}

	sourcesFiles, err := m.findSourcesFiles()
	if err != nil {
		m.logger.WithError(err).Error("Failed to find sources files")
		return repositories, err
	}

	// Parse .list files
	for _, file := range listFiles {
		repos, err := m.parseSourcesList(file)
		if err != nil {
			m.logger.Warnf("Error parsing %s: %v", file, err)
			continue
		}
		repositories = append(repositories, repos...)
	}

	// Parse modern DEB822 format (.sources files)
	for _, file := range sourcesFiles {
		repos, err := m.parseDEB822Sources(file)
		if err != nil {
			m.logger.Warnf("Error parsing %s: %v", file, err)
			continue
		}
		repositories = append(repositories, repos...)
	}

	return repositories, nil
}

// findListFiles finds all .list files in common locations
func (m *APTManager) findListFiles() ([]string, error) {
	var listFiles []string

	// Add main sources.list file
	listFiles = append(listFiles, "/etc/apt/sources.list")

	// Add .list files from sources.list.d
	sourcesDir := "/etc/apt/sources.list.d"
	if entries, err := os.ReadDir(sourcesDir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}

			if strings.HasSuffix(entry.Name(), ".list") {
				fullPath := filepath.Join(sourcesDir, entry.Name())
				listFiles = append(listFiles, fullPath)
			}
		}
	} else {
		m.logger.WithError(err).WithField("path", sourcesDir).Warn("Failed to read sources directory for .list files")
	}

	return listFiles, nil
}

// findSourcesFiles finds all .sources files in common locations
func (m *APTManager) findSourcesFiles() ([]string, error) {
	var sourcesFiles []string

	// Add .sources files from sources.list.d
	sourcesDir := "/etc/apt/sources.list.d"
	if entries, err := os.ReadDir(sourcesDir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}

			if strings.HasSuffix(entry.Name(), ".sources") {
				fullPath := filepath.Join(sourcesDir, entry.Name())
				sourcesFiles = append(sourcesFiles, fullPath)
			}
		}
	} else {
		m.logger.WithError(err).WithField("path", sourcesDir).Warn("Failed to read sources directory for .sources files")
	}

	return sourcesFiles, nil
}

// parseSourcesList parses traditional APT sources.list files
func (m *APTManager) parseSourcesList(filename string) ([]models.Repository, error) {
	var repositories []models.Repository

	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip comments and empty lines
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse repository line (deb or deb-src)
		if strings.HasPrefix(line, "deb") {
			repo := m.parseSourceLine(line)
			if repo != nil {
				repositories = append(repositories, *repo)
			}
		}
	}

	return repositories, scanner.Err()
}

// parseSourceLine parses a single APT source line
func (m *APTManager) parseSourceLine(line string) *models.Repository {
	fields := slices.Collect(strings.FieldsSeq(line))
	if len(fields) < 4 {
		return nil
	}

	repoType := fields[0]
	var url, distribution, components string
	var fieldIndex int

	// Handle modern format with options like [signed-by=...]
	if len(fields) > 1 && strings.HasPrefix(fields[1], "[") {
		// Find the end of options
		optionsEnd := 1
		for i := 1; i < len(fields); i++ {
			if strings.HasSuffix(fields[i], "]") {
				optionsEnd = i
				break
			}
		}
		fieldIndex = optionsEnd + 1
	} else {
		fieldIndex = 1
	}

	if fieldIndex+2 >= len(fields) {
		return nil
	}

	url = fields[fieldIndex]
	distribution = fields[fieldIndex+1]
	if fieldIndex+2 < len(fields) {
		components = strings.Join(fields[fieldIndex+2:], " ")
	}

	// Skip if URL doesn't look valid
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") && !strings.HasPrefix(url, "ftp://") {
		return nil
	}

	// Skip if distribution is empty or looks malformed
	if distribution == "" || strings.Contains(distribution, "[") {
		return nil
	}

	// Determine repository name
	repoName := generateRepoName(url, distribution, components)

	// Check if repository uses HTTPS
	isSecure := strings.HasPrefix(url, "https://")

	return &models.Repository{
		Name:         repoName,
		URL:          url,
		Distribution: distribution,
		Components:   components,
		RepoType:     repoType,
		IsEnabled:    true,
		IsSecure:     isSecure,
	}
}

// parseDEB822Sources parses modern DEB822 format sources files
func (m *APTManager) parseDEB822Sources(filename string) ([]models.Repository, error) {
	var repositories []models.Repository

	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var currentEntry map[string]string
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip comments
		if strings.HasPrefix(line, "#") {
			continue
		}

		// Empty line indicates end of entry
		if line == "" {
			if currentEntry != nil {
				repos := m.processDEB822Entry(currentEntry)
				repositories = append(repositories, repos...)
				currentEntry = nil
			}
			continue
		}

		// Parse key-value pairs
		if strings.Contains(line, ":") {
			if currentEntry == nil {
				currentEntry = make(map[string]string)
			}

			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				currentEntry[key] = value
			}
		}
	}

	// Process last entry if file doesn't end with empty line
	if currentEntry != nil {
		repos := m.processDEB822Entry(currentEntry)
		repositories = append(repositories, repos...)
	}

	return repositories, scanner.Err()
}

// processDEB822Entry processes a single DEB822 repository entry
func (m *APTManager) processDEB822Entry(entry map[string]string) []models.Repository {
	var repositories []models.Repository

	enabled := entry["Enabled"]
	if enabled != "yes" {
		return repositories
	}

	types := entry["Types"]
	uris := entry["URIs"]
	suites := entry["Suites"]
	components := entry["Components"]
	name := entry["X-Repolib-Name"]

	if uris == "" || suites == "" {
		return repositories
	}

	// Split multiple values
	uriList := slices.Collect(strings.FieldsSeq(uris))
	suiteList := slices.Collect(strings.FieldsSeq(suites))

	for _, uri := range uriList {
		// Skip invalid URIs
		if !strings.HasPrefix(uri, "http://") && !strings.HasPrefix(uri, "https://") && !strings.HasPrefix(uri, "ftp://") {
			continue
		}

		for _, suite := range suiteList {
			if suite == "" {
				continue
			}

			// Generate repository name
			repoName := name
			if repoName == "" {
				repoName = generateRepoName(uri, suite, components)
			} else {
				repoName = strings.ToLower(strings.ReplaceAll(repoName, " ", "-"))
			}

			// Determine repo type
			repoType := "deb"
			if strings.Contains(types, "deb-src") && !strings.Contains(types, "deb ") {
				repoType = "deb-src"
			}

			isSecure := strings.HasPrefix(uri, "https://")

			repositories = append(repositories, models.Repository{
				Name:         repoName,
				URL:          uri,
				Distribution: suite,
				Components:   components,
				RepoType:     repoType,
				IsEnabled:    true,
				IsSecure:     isSecure,
			})
		}
	}

	return repositories
}
