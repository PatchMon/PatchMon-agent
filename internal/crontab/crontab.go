package crontab

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"patchmon-agent/internal/config"
)

// Manager handles crontab operations
type Manager struct {
	logger *logrus.Logger
}

// New creates a new crontab manager
func New(logger *logrus.Logger) *Manager {
	return &Manager{
		logger: logger,
	}
}

// UpdateSchedule updates the cron schedule with the given interval and executable path
func (m *Manager) UpdateSchedule(updateInterval int, executablePath string) error {
	if updateInterval <= 0 {
		return fmt.Errorf("invalid update interval: %d", updateInterval)
	}

	// Generate crontab entry
	expectedEntry := m.generateCronEntry(updateInterval, executablePath)

	// Check if current entry is up to date
	if currentEntry := m.GetEntry(); currentEntry == expectedEntry {
		m.logger.Infof("Crontab is already up to date (interval: %d minutes)", updateInterval)
		return nil
	}

	m.logger.Infof("Setting update interval to %d minutes", updateInterval)

	// Write crontab file
	if err := os.WriteFile(config.CronFilePath, []byte(expectedEntry+"\n"), 0644); err != nil {
		return fmt.Errorf("failed to update crontab file: %w", err)
	}

	m.logger.Info("Crontab updated successfully")
	return nil
}

// GetEntry returns the current cron entry
func (m *Manager) GetEntry() string {
	if data, err := os.ReadFile(config.CronFilePath); err == nil {
		lines := strings.Split(strings.TrimSpace(string(data)), "\n")

		// Filter out empty lines and comments
		var validLines []string
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" && !strings.HasPrefix(line, "#") {
				validLines = append(validLines, line)
			}
		}

		switch len(validLines) {
		case 0:
			return ""
		case 1:
			return validLines[0]
		default:
			m.logger.Warnf("Multiple cron entries found in %s, expected only one", config.CronFilePath)
			return validLines[0]
		}
	}
	return ""
}

// Remove removes the PatchMon agent's cron file
func (m *Manager) Remove() error {
	if err := os.Remove(config.CronFilePath); err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist, which is fine
			return nil
		}
		return fmt.Errorf("failed to remove cron file: %w", err)
	}

	m.logger.Info("Removed cron file")
	return nil
}

// generateCronEntry generates a cron entry for the given interval and executable
func (m *Manager) generateCronEntry(updateInterval int, executablePath string) string {
	if updateInterval == 60 {
		// Hourly updates - use current minute to spread load
		currentMinute := time.Now().Minute()
		return fmt.Sprintf("%d * * * * root %s update >/dev/null 2>&1", currentMinute, executablePath)
	}

	// Custom interval updates
	return fmt.Sprintf("*/%d * * * * root %s update >/dev/null 2>&1", updateInterval, executablePath)
}
