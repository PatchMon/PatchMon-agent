package packages

import (
	"testing"

	"patchmon-agent/pkg/models"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestAPTManager_parseInstalledPackages(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	manager := NewAPTManager(logger)

	tests := []struct {
		name     string
		input    string
		expected map[string]string
	}{
		{
			name: "valid single package",
			input: `vim 2:8.2.3995-1ubuntu2.17
`,
			expected: map[string]string{
				"vim": "2:8.2.3995-1ubuntu2.17",
			},
		},
		{
			name: "multiple packages",
			input: `vim 2:8.2.3995-1ubuntu2.17
libc6 2.35-0ubuntu3.8
bash 5.1-6ubuntu1.1
`,
			expected: map[string]string{
				"vim":   "2:8.2.3995-1ubuntu2.17",
				"libc6": "2.35-0ubuntu3.8",
				"bash":  "5.1-6ubuntu1.1",
			},
		},
		{
			name:     "empty input",
			input:    "",
			expected: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := manager.parseInstalledPackages(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAPTManager_parseAPTUpgrade(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	manager := NewAPTManager(logger)

	tests := []struct {
		name     string
		input    string
		expected []models.Package
	}{
		{
			name:  "standard upgrade",
			input: `Inst vim [2:8.2.3995-1ubuntu2.16] (2:8.2.3995-1ubuntu2.17 Ubuntu:22.04/jammy-updates [amd64])`,
			expected: []models.Package{
				{
					Name:             "vim",
					CurrentVersion:   "2:8.2.3995-1ubuntu2.16",
					AvailableVersion: "2:8.2.3995-1ubuntu2.17",
					NeedsUpdate:      true,
					IsSecurityUpdate: false,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := manager.parseAPTUpgrade(tt.input)
			assert.Equal(t, len(tt.expected), len(result))
		})
	}
}
