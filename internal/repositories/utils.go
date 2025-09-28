package repositories

import (
	"fmt"
	"strings"
)

// generateRepoName generates a meaningful repository name
func generateRepoName(url, distribution, components string) string {
	// Extract meaningful name from URL
	if strings.Contains(url, "archive.ubuntu.com") {
		return fmt.Sprintf("ubuntu-%s", distribution)
	}
	if strings.Contains(url, "security.ubuntu.com") {
		return fmt.Sprintf("ubuntu-%s-security", distribution)
	}
	if strings.Contains(url, "apt.pop-os.org/ubuntu") {
		return fmt.Sprintf("pop-os-ubuntu-%s", distribution)
	}
	if strings.Contains(url, "apt.pop-os.org/release") {
		return fmt.Sprintf("pop-os-release-%s", distribution)
	}
	if strings.Contains(url, "apt.pop-os.org/proprietary") {
		return fmt.Sprintf("pop-os-apps-%s", distribution)
	}
	if strings.Contains(url, "deb.nodesource.com") {
		return fmt.Sprintf("nodesource-%s", distribution)
	}
	if strings.Contains(url, "packages.microsoft.com") {
		return fmt.Sprintf("microsoft-%s", distribution)
	}
	if strings.Contains(url, "download.docker.com") {
		return fmt.Sprintf("docker-%s", distribution)
	}

	// Extract domain name as fallback
	parts := strings.Split(url, "/")
	if len(parts) >= 3 {
		domain := strings.Split(parts[2], ":")[0] // Remove port if present
		repoName := fmt.Sprintf("%s-%s", domain, distribution)

		// Add component suffix if relevant
		if strings.Contains(components, "security") && !strings.Contains(distribution, "security") {
			repoName += "-security"
		} else if strings.Contains(components, "updates") && !strings.Contains(distribution, "updates") {
			repoName += "-updates"
		} else if strings.Contains(components, "backports") && !strings.Contains(distribution, "backports") {
			repoName += "-backports"
		}

		return repoName
	}

	// Final fallback
	return distribution

}
