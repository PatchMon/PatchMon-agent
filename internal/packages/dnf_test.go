package packages

import (
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestDNFManager_parseInstalledPackages(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	manager := NewDNFManager(logger)

	tests := []struct {
		name     string
		input    string
		expected map[string]string
	}{
		{
			name: "valid packages",
			input: `Installed Packages
vim-enhanced.x86_64                  2:8.2.2637-20.el9_1                  @baseos
bash.x86_64                          5.1.8-6.el9_1                        @baseos`,
			expected: map[string]string{
				"vim-enhanced.x86_64": "2:8.2.2637-20.el9_1",
				"bash.x86_64":         "5.1.8-6.el9_1",
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

func TestDNFManager_parseUpgradablePackages(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	manager := NewDNFManager(logger)

	tests := []struct {
		name     string
		input    string
		pkgMgr   string
		expected int
	}{
		{
			name: "upgradable packages",
			input: `kernel.x86_64                     5.14.0-284.30.1.el9_2           baseos
systemd.x86_64                    252-14.el9_2.2                  baseos`,
			pkgMgr:   "dnf",
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := manager.parseUpgradablePackages(tt.input, tt.pkgMgr)
			assert.Equal(t, tt.expected, len(result))
		})
	}
}
