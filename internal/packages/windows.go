package packages

import (
	"encoding/json"
	"os/exec"
	"strings"

	"patchmon-agent/pkg/models"

	"github.com/sirupsen/logrus"
)

// WindowsManager handles Windows Update package information collection
type WindowsManager struct {
	logger *logrus.Logger
}

// NewWindowsManager creates a new Windows package manager
func NewWindowsManager(logger *logrus.Logger) *WindowsManager {
	return &WindowsManager{
		logger: logger,
	}
}

// WindowsUpdate represents a Windows Update item
type windowsUpdate struct {
	Title            string `json:"Title"`
	Description      string `json:"Description"`
	UpdateID         string `json:"UpdateID"`
	KBArticleIDs     string `json:"KBArticleIDs"`
	IsDownloaded     bool   `json:"IsDownloaded"`
	IsInstalled      bool   `json:"IsInstalled"`
	IsMandatory      bool   `json:"IsMandatory"`
	IsSecurityUpdate bool   `json:"IsSecurityUpdate"`
	Categories       string `json:"Categories"`
}

// GetPackages gets package information from Windows Update
func (m *WindowsManager) GetPackages() []models.Package {
	m.logger.Debug("Collecting Windows Update information...")

	// PowerShell script to get Windows Updates
	// This uses the Microsoft.Update.Session COM interface
	psScript := `
$ErrorActionPreference = "Stop"
$updateSession = New-Object -ComObject Microsoft.Update.Session
$updateSearcher = $updateSession.CreateUpdateSearcher()
$searchResult = $updateSearcher.Search("IsInstalled=0 and IsHidden=0")

$updates = @()
foreach ($update in $searchResult.Updates) {
    $kbArticles = ""
    if ($update.KBArticleIDs.Count -gt 0) {
        $kbArticles = $update.KBArticleIDs -join ", "
    }
    
    $categories = ""
    if ($update.Categories.Count -gt 0) {
        $categoryNames = $update.Categories | ForEach-Object { $_.Name }
        $categories = $categoryNames -join ", "
    }
    
    $isSecurity = $false
    if ($categories -like "*Security*" -or $categories -like "*Critical*") {
        $isSecurity = $true
    }
    
    $updateObj = @{
        Title = $update.Title
        Description = $update.Description
        UpdateID = $update.Identity.UpdateID
        KBArticleIDs = $kbArticles
        IsDownloaded = $update.IsDownloaded
        IsInstalled = $update.IsInstalled
        IsMandatory = $update.IsMandatory
        IsSecurityUpdate = $isSecurity
        Categories = $categories
    }
    $updates += $updateObj
}

$updates | ConvertTo-Json -Compress
`

	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", psScript)
	output, err := cmd.Output()
	if err != nil {
		m.logger.WithError(err).Warn("Failed to query Windows Update (may require admin privileges or no updates available)")
		// Return empty list if we can't query Windows Update
		return []models.Package{}
	}

	updatesJSON := strings.TrimSpace(string(output))
	if updatesJSON == "" || updatesJSON == "[]" {
		m.logger.Debug("No Windows Updates available")
		return []models.Package{}
	}

	// Parse JSON output
	var updates []windowsUpdate
	if err := json.Unmarshal([]byte(updatesJSON), &updates); err != nil {
		m.logger.WithError(err).Warn("Failed to parse Windows Update JSON")
		return []models.Package{}
	}

	// Convert to Package models
	var packages []models.Package
	for _, update := range updates {
		// Use UpdateID or Title as package name
		name := update.UpdateID
		if name == "" {
			name = update.Title
		}

		// Extract version if available (often in KB number)
		currentVersion := "installed"
		availableVersion := ""
		if update.KBArticleIDs != "" {
			availableVersion = "KB" + update.KBArticleIDs
		} else {
			availableVersion = update.UpdateID
		}

		packages = append(packages, models.Package{
			Name:             name,
			CurrentVersion:   currentVersion,
			AvailableVersion: availableVersion,
			NeedsUpdate:      true,
			IsSecurityUpdate: update.IsSecurityUpdate,
		})
	}

	m.logger.WithField("count", len(packages)).Info("Found Windows Updates")
	return packages
}

