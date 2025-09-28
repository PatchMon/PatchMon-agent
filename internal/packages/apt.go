package packages

import (
	"bufio"
	"os/exec"
	"slices"
	"strings"

	"patchmon-agent/pkg/models"

	"github.com/sirupsen/logrus"
)

// APTManager handles APT package information collection
type APTManager struct {
	logger *logrus.Logger
}

// NewAPTManager creates a new APT package manager
func NewAPTManager(logger *logrus.Logger) *APTManager {
	return &APTManager{
		logger: logger,
	}
}

// detectPackageManager detects whether to use apt or apt-get
func (m *APTManager) detectPackageManager() string {
	// Prefer apt over apt-get for modern Debian-based systems
	packageManager := "apt"
	if _, err := exec.LookPath("apt"); err != nil {
		packageManager = "apt-get"
	}
	return packageManager
}

// GetPackages gets package information for APT-based systems
func (m *APTManager) GetPackages() ([]models.Package, error) {
	var packages []models.Package

	// Determine package manager
	packageManager := m.detectPackageManager()

	// Update package lists using detected package manager
	m.logger.Debug("Updating package lists...")
	updateCmd := exec.Command(packageManager, "update", "-qq")

	if err := updateCmd.Run(); err != nil {
		m.logger.Warnf("Failed to update package lists: %v", err)
	}

	// Get upgradable packages using simulation
	m.logger.Debug("Getting upgradable packages...")
	upgradeCmd := exec.Command(packageManager, "-s", "-o", "Debug::NoLocking=1", "upgrade")

	upgradeOutput, err := upgradeCmd.Output()
	if err != nil {
		m.logger.Warnf("Failed to get upgrade simulation: %v", err)
	} else {
		upgradablePackages := m.parseAPTUpgrade(string(upgradeOutput))
		packages = append(packages, upgradablePackages...)
	}

	// Get installed packages that are up to date
	m.logger.Debug("Getting installed packages...")
	installedCmd := exec.Command("dpkg-query", "-W", "-f", "${Package} ${Version}\n")
	installedOutput, err := installedCmd.Output()
	if err != nil {
		m.logger.Warnf("Failed to get installed packages: %v", err)
	} else {
		installedPackages := m.parseInstalledPackages(string(installedOutput), packages)
		packages = append(packages, installedPackages...)
	}

	return packages, nil
}

// parseAPTUpgrade parses apt/apt-get upgrade simulation output
func (m *APTManager) parseAPTUpgrade(output string) []models.Package {
	var packages []models.Package

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Look for lines starting with "Inst"
		if !strings.HasPrefix(line, "Inst ") {
			continue
		}

		// Parse the line: Inst package [current_version] (new_version source)
		fields := slices.Collect(strings.FieldsSeq(line))
		if len(fields) < 4 {
			continue
		}

		packageName := fields[1]

		// Extract current version (in brackets)
		var currentVersion string
		for i, field := range fields {
			if strings.HasPrefix(field, "[") && strings.HasSuffix(field, "]") {
				currentVersion = strings.Trim(field, "[]")
				break
			} else if after, found := strings.CutPrefix(field, "["); found {
				// Multi-word version, continue until we find the closing bracket
				versionParts := []string{after}
				for j := i + 1; j < len(fields); j++ {
					if strings.HasSuffix(fields[j], "]") {
						versionParts = append(versionParts, strings.TrimSuffix(fields[j], "]"))
						break
					} else {
						versionParts = append(versionParts, fields[j])
					}
				}
				currentVersion = strings.Join(versionParts, " ")
				break
			}
		}

		// Extract available version (in parentheses)
		var availableVersion string
		for _, field := range fields {
			if after, found := strings.CutPrefix(field, "("); found {
				availableVersion = after
				break
			}
		}

		// Check if it's a security update
		isSecurityUpdate := strings.Contains(strings.ToLower(line), "security")

		if packageName != "" && currentVersion != "" && availableVersion != "" {
			packages = append(packages, models.Package{
				Name:             packageName,
				CurrentVersion:   currentVersion,
				AvailableVersion: availableVersion,
				NeedsUpdate:      true,
				IsSecurityUpdate: isSecurityUpdate,
			})
		}
	}

	return packages
}

// parseInstalledPackages parses dpkg-query output for installed packages
func (m *APTManager) parseInstalledPackages(output string, upgradablePackages []models.Package) []models.Package {
	var packages []models.Package

	// Create a map of upgradable packages for quick lookup
	upgradableMap := make(map[string]bool)
	for _, pkg := range upgradablePackages {
		upgradableMap[pkg.Name] = true
	}

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, " ", 2)
		if len(parts) != 2 {
			continue
		}

		packageName := parts[0]
		version := parts[1]

		// Skip if already in upgradable list
		if upgradableMap[packageName] {
			continue
		}

		packages = append(packages, models.Package{
			Name:             packageName,
			CurrentVersion:   version,
			NeedsUpdate:      false,
			IsSecurityUpdate: false,
		})
	}

	return packages
}
