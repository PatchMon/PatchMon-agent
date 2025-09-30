package packages

import (
	"bufio"
	"os/exec"
	"slices"
	"strings"

	"patchmon-agent/pkg/models"

	"github.com/sirupsen/logrus"
)

// DNFManager handles dnf/yum package information collection
type DNFManager struct {
	logger *logrus.Logger
}

// NewDNFManager creates a new DNF package manager
func NewDNFManager(logger *logrus.Logger) *DNFManager {
	return &DNFManager{
		logger: logger,
	}
}

// detectPackageManager detects whether to use dnf or yum
func (m *DNFManager) detectPackageManager() string {
	// Prefer dnf over yum for modern RHEL-based systems
	packageManager := "dnf"
	if _, err := exec.LookPath("dnf"); err != nil {
		// Fall back to yum if dnf is not available (legacy systems)
		packageManager = "yum"
	}
	return packageManager
}

// GetPackages gets package information for RHEL-based systems
func (m *DNFManager) GetPackages() ([]models.Package, error) {
	var packages []models.Package

	// Determine package manager
	packageManager := m.detectPackageManager()

	m.logger.Debugf("Using package manager: %s", packageManager)

	// Get upgradable packages
	m.logger.Debug("Getting upgradable packages...")
	checkCmd := exec.Command(packageManager, "check-update")
	checkOutput, _ := checkCmd.Output() // This command returns exit code 100 when updates are available

	if len(checkOutput) > 0 {
		m.logger.Debug("Parsing DNF/yum check-update output...")
		upgradablePackages := m.parseDnfCheckUpdate(string(checkOutput), packageManager)
		m.logger.Debugf("Found %d upgradable packages", len(upgradablePackages))
		packages = append(packages, upgradablePackages...)
	} else {
		m.logger.Debug("No updates available")
	}

	// Get installed packages
	m.logger.Debug("Getting installed packages...")
	listCmd := exec.Command(packageManager, "list", "installed")
	listOutput, err := listCmd.Output()
	if err != nil {
		m.logger.Warnf("Failed to get installed packages: %v", err)
	} else {
		m.logger.Debug("Parsing installed packages...")
		installedPackages := m.parseDnfList(string(listOutput), packages)
		m.logger.Debugf("Found %d installed packages", len(installedPackages))
		packages = append(packages, installedPackages...)
	}

	m.logger.Debugf("Total packages collected: %d", len(packages))
	return packages, nil
}

// parseDnfCheckUpdate parses dnf/yum check-update output
func (m *DNFManager) parseDnfCheckUpdate(output string, packageManager string) []models.Package {
	var packages []models.Package

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip header lines and empty lines
		if line == "" || strings.Contains(line, "Loaded plugins") ||
			strings.Contains(line, "Last metadata") || strings.HasPrefix(line, "Loading") {
			continue
		}

		fields := slices.Collect(strings.FieldsSeq(line))
		if len(fields) < 3 {
			continue
		}

		packageName := fields[0]
		availableVersion := fields[1]
		repo := fields[2]

		// Get current version
		getCurrentCmd := exec.Command(packageManager, "list", "installed", packageName)
		getCurrentOutput, err := getCurrentCmd.Output()
		var currentVersion string
		if err == nil {
			for currentLine := range strings.SplitSeq(string(getCurrentOutput), "\n") {
				if strings.Contains(currentLine, packageName) && !strings.Contains(currentLine, "Installed") {
					currentFields := slices.Collect(strings.FieldsSeq(currentLine))
					if len(currentFields) >= 2 {
						currentVersion = currentFields[1]
						break
					}
				}
			}
		}

		isSecurityUpdate := strings.Contains(strings.ToLower(repo), "security")

		packages = append(packages, models.Package{
			Name:             packageName,
			CurrentVersion:   currentVersion,
			AvailableVersion: availableVersion,
			NeedsUpdate:      true,
			IsSecurityUpdate: isSecurityUpdate,
		})
	}

	return packages
}

// parseDnfList parses dnf/yum list installed output
func (m *DNFManager) parseDnfList(output string, upgradablePackages []models.Package) []models.Package {
	var packages []models.Package

	// Create a map of upgradable packages for quick lookup
	upgradableMap := make(map[string]bool)
	for _, pkg := range upgradablePackages {
		upgradableMap[pkg.Name] = true
	}

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip header lines and empty lines
		if line == "" || strings.Contains(line, "Loaded plugins") ||
			strings.Contains(line, "Installed Packages") {
			continue
		}

		fields := slices.Collect(strings.FieldsSeq(line))
		if len(fields) < 2 {
			continue
		}

		packageName := fields[0]
		version := fields[1]

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
