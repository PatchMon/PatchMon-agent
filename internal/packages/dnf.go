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
func (m *DNFManager) GetPackages() []models.Package {
	// Determine package manager
	packageManager := m.detectPackageManager()

	m.logger.WithField("manager", packageManager).Debug("Using package manager")

	// Get installed packages
	m.logger.Debug("Getting installed packages...")
	listCmd := exec.Command(packageManager, "list", "installed")
	listOutput, err := listCmd.Output()
	var installedPackages map[string]string
	if err != nil {
		m.logger.WithError(err).Warn("Failed to get installed packages")
		installedPackages = make(map[string]string)
	} else {
		m.logger.Debug("Parsing installed packages...")
		installedPackages = m.parseInstalledPackages(string(listOutput))
		m.logger.WithField("count", len(installedPackages)).Debug("Found installed packages")
	}

	// Get upgradable packages
	m.logger.Debug("Getting upgradable packages...")
	checkCmd := exec.Command(packageManager, "check-update")
	checkOutput, _ := checkCmd.Output() // This command returns exit code 100 when updates are available

	var upgradablePackages []models.Package
	if len(checkOutput) > 0 {
		m.logger.Debug("Parsing DNF/yum check-update output...")
		upgradablePackages = m.parseUpgradablePackages(string(checkOutput), packageManager)
		m.logger.WithField("count", len(upgradablePackages)).Debug("Found upgradable packages")
	} else {
		m.logger.Debug("No updates available")
		upgradablePackages = []models.Package{}
	}

	// Merge and deduplicate packages
	packages := CombinePackageData(installedPackages, upgradablePackages)
	m.logger.WithField("total", len(packages)).Debug("Total packages collected")

	return packages
}

// parseUpgradablePackages parses dnf/yum check-update output
func (m *DNFManager) parseUpgradablePackages(output string, packageManager string) []models.Package {
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

// parseInstalledPackages parses dnf/yum list installed output and returns a map of package name to version
func (m *DNFManager) parseInstalledPackages(output string) map[string]string {
	installedPackages := make(map[string]string)

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
		installedPackages[packageName] = version
	}

	return installedPackages
}
