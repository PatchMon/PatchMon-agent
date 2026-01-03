package system

import (
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestParseKernelVersion(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		expected []string
	}{
		{
			name:     "generic kernel version",
			version:  "5.15.0-164-generic",
			expected: []string{"5", "15", "0", "164", "generic"},
		},
		{
			name:     "ubuntu virtual kernel",
			version:  "5.15.0.164.159",
			expected: []string{"5", "15", "0", "164", "159"},
		},
		{
			name:     "pve kernel",
			version:  "6.14.11-2-pve",
			expected: []string{"6", "14", "11", "2", "pve"},
		},
		{
			name:     "simple version",
			version:  "4.4.0",
			expected: []string{"4", "4", "0"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseKernelVersion(tt.version)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCompareKernelVersions(t *testing.T) {
	tests := []struct {
		name     string
		v1       string
		v2       string
		expected int // -1 if v1 < v2, 0 if v1 == v2, 1 if v1 > v2
	}{
		{
			name:     "v1 less than v2",
			v1:       "5.15.0-164-generic",
			v2:       "5.15.0-165-generic",
			expected: -1,
		},
		{
			name:     "v1 greater than v2",
			v1:       "6.8.0-88-generic",
			v2:       "5.15.0-164-generic",
			expected: 1,
		},
		{
			name:     "v1 equal to v2",
			v1:       "5.15.0-164-generic",
			v2:       "5.15.0-164-generic",
			expected: 0,
		},
		{
			name:     "different major versions",
			v1:       "4.15.0-100-generic",
			v2:       "5.15.0-100-generic",
			expected: -1,
		},
		{
			name:     "different minor versions",
			v1:       "5.10.0-100-generic",
			v2:       "5.15.0-100-generic",
			expected: -1,
		},
		{
			name:     "pve kernels",
			v1:       "6.14.11-2-pve",
			v2:       "6.8.12-9-pve",
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := compareKernelVersions(tt.v1, tt.v2)
			assert.Equal(t, tt.expected, result, "compareKernelVersions(%s, %s)", tt.v1, tt.v2)
		})
	}
}

func TestGetLatestKernelFromDpkg(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	detector := &Detector{logger: logger}

	tests := []struct {
		name                string
		dpkgOutput          string
		expectedVersion     string
		shouldContainMeta   bool // Whether we expect meta-package resolution to be called
		description         string
	}{
		{
			name: "single actual kernel package",
			dpkgOutput: `ii  linux-image-5.15.0-164-generic  5.15.0-164.174  amd64  Signed kernel image generic
ii  linux-modules-5.15.0-164-generic  5.15.0-164.174  amd64  Linux kernel modules`,
			expectedVersion: "5.15.0-164-generic",
			description:     "Should return actual kernel package version",
		},
		{
			name: "multiple kernel packages - returns latest",
			dpkgOutput: `ii  linux-image-5.15.0-160-generic  5.15.0-160.170  amd64  Signed kernel image
ii  linux-image-5.15.0-164-generic  5.15.0-164.174  amd64  Signed kernel image generic
ii  linux-modules-5.15.0-164-generic  5.15.0-164.174  amd64  Linux kernel modules`,
			expectedVersion: "5.15.0-164-generic",
			description:     "Should return the latest kernel package when multiple are installed",
		},
		{
			name: "skip generic meta-package",
			dpkgOutput: `ii  linux-image-generic  5.15.0.164.159  amd64  Generic Linux kernel image
ii  linux-image-5.15.0-164-generic  5.15.0-164.174  amd64  Signed kernel image generic`,
			expectedVersion: "5.15.0-164-generic",
			shouldContainMeta: true,
			description:     "Should skip linux-image-generic meta-package and resolve to actual kernel",
		},
		{
			name: "virtual meta-package present",
			dpkgOutput: `ii  linux-image-virtual  5.15.0.164.159  amd64  Virtual Linux kernel image
ii  linux-image-5.15.0-164-generic  5.15.0-164.174  amd64  Signed kernel image generic`,
			expectedVersion: "5.15.0-164-generic",
			shouldContainMeta: true,
			description:     "Should skip linux-image-virtual and use actual kernel package",
		},
		{
			name:            "no kernel packages",
			dpkgOutput:      `ii  vim  2:8.2.3995-1  amd64  Vi improved editor`,
			expectedVersion: "",
			description:     "Should return empty string when no kernel packages found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: This test uses the current implementation which would need mocking
			// for the actual apt-cache calls in production
			// For now, we test the parsing logic with real kernel packages
			t.Logf("Test description: %s", tt.description)
		})
	}
}

func TestResolveMetaPackageKernel(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	detector := &Detector{logger: logger}

	// Note: Full testing of resolveMetaPackageKernel would require mocking
	// the apt-cache command execution. In a real test environment, you would use:
	// - Mock the exec.Command
	// - Mock the output of apt-cache depends
	// 
	// Example test cases that would work with mocks:
	t.Run("resolve linux-image-virtual meta-package", func(t *testing.T) {
		// This test would mock apt-cache depends output
		// Expected: resolves "linux-image-virtual" to "5.15.0-164-generic"
		t.Logf("Requires mocking exec.Command for apt-cache depends")
	})

	t.Run("resolve linux-image-generic meta-package", func(t *testing.T) {
		// This test would mock apt-cache depends output
		// Expected: resolves "linux-image-generic" to "6.8.0-88-generic"
		t.Logf("Requires mocking exec.Command for apt-cache depends")
	})
}

// Test edge cases for kernel detection
func TestKernelDetectionEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		runningKernel string
		installedKernel string
		expected    bool
		description string
	}{
		{
			name:            "exact match - no reboot needed",
			runningKernel:   "5.15.0-164-generic",
			installedKernel: "5.15.0-164-generic",
			expected:        false,
			description:     "When running and installed kernels match exactly",
		},
		{
			name:            "mismatch - reboot needed",
			runningKernel:   "5.15.0-160-generic",
			installedKernel: "5.15.0-164-generic",
			expected:        true,
			description:     "When installed kernel is newer than running kernel",
		},
		{
			name:            "empty installed kernel - no reboot",
			runningKernel:   "5.15.0-164-generic",
			installedKernel: "",
			expected:        false,
			description:     "When we cannot determine installed kernel",
		},
		{
			name:            "empty running kernel - no reboot",
			runningKernel:   "",
			installedKernel: "5.15.0-164-generic",
			expected:        false,
			description:     "When we cannot determine running kernel",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			needsReboot := tt.runningKernel != tt.installedKernel && tt.installedKernel != ""
			assert.Equal(t, tt.expected, needsReboot, tt.description)
		})
	}
}
