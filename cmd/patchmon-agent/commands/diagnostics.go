package commands

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"patchmon-agent/internal/crontab"
	"patchmon-agent/internal/utils"
	"patchmon-agent/internal/version"

	"github.com/spf13/cobra"
)

// diagnosticsCmd represents the diagnostics command
var diagnosticsCmd = &cobra.Command{
	Use:   "diagnostics",
	Short: "Show detailed system diagnostics",
	Long:  "Display comprehensive diagnostic information about the agent, system, and configuration.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return showDiagnostics()
	},
}

func showDiagnostics() error {
	cfg := cfgManager.GetConfig()

	fmt.Printf("PatchMon Agent Diagnostics v%s\n", version.Version)
	fmt.Printf("=====================================\n\n")

	// System Information
	fmt.Printf("=== System Information ===\n")
	fmt.Printf("OS: %s\n", runtime.GOOS)
	fmt.Printf("Architecture: %s\n", runtime.GOARCH)

	if kernelVersion, err := utils.GetKernelVersion(); err == nil {
		fmt.Printf("Kernel: %s\n", kernelVersion)
	}

	if hostname, err := os.Hostname(); err == nil {
		fmt.Printf("Hostname: %s\n", hostname)
	}

	fmt.Printf("\n")

	// Agent Information
	fmt.Printf("=== Agent Information ===\n")
	fmt.Printf("Version: %s\n", version.Version)

	if execPath, err := os.Executable(); err == nil {
		fmt.Printf("Executable Path: %s\n", execPath)

		if stat, err := os.Stat(execPath); err == nil {
			fmt.Printf("Executable Size: %d bytes\n", stat.Size())
			fmt.Printf("Last Modified: %s\n", stat.ModTime().Format(time.RFC3339))
		}
	}

	fmt.Printf("Config File: %s\n", cfgManager.GetConfigFile())
	fmt.Printf("Credentials File: %s\n", cfg.CredentialsFile)
	fmt.Printf("Log File: %s\n", cfg.LogFile)
	fmt.Printf("Log Level: %s\n", cfg.LogLevel)
	fmt.Printf("\n")

	// Configuration Status
	fmt.Printf("=== Configuration Status ===\n")
	configFile := cfgManager.GetConfigFile()
	if _, err := os.Stat(configFile); err == nil {
		fmt.Printf("Config file exists: Yes\n")
		if stat, err := os.Stat(configFile); err == nil {
			fmt.Printf("Config file size: %d bytes\n", stat.Size())
		}
	} else {
		fmt.Printf("Config file exists: No (using defaults)\n")
	}
	fmt.Printf("\n")

	// Credentials Status
	fmt.Printf("=== Credentials Status ===\n")
	if stat, err := os.Stat(cfg.CredentialsFile); err == nil {
		fmt.Printf("Credentials file exists: Yes\n")
		fmt.Printf("File size: %d bytes\n", stat.Size())
		fmt.Printf("File permissions: %o\n", stat.Mode().Perm())
	} else {
		fmt.Printf("Credentials file exists: No\n")
	}
	fmt.Printf("\n")

	// Crontab Status
	fmt.Printf("=== Crontab Status ===\n")
	cronManager := crontab.New(logger)
	if crontabEntry := cronManager.GetEntry(); crontabEntry != "" {
		fmt.Printf("Crontab entry:\n")
		fmt.Printf("  %s\n", crontabEntry)
	} else {
		fmt.Printf("No crontab entries found\n")
	}
	fmt.Printf("\n")

	// Network Connectivity & API Credentials
	fmt.Printf("=== Network Connectivity & API Credentials ===\n")
	fmt.Printf("Server URL: %s\n", cfg.PatchmonServer)

	// Extract hostname and port from server URL for basic connectivity test
	serverURL := cfg.PatchmonServer
	var serverHost, serverPort string

	trimmed := strings.TrimPrefix(serverURL, "http://")
	trimmed = strings.TrimPrefix(trimmed, "https://")
	hostPort := strings.SplitN(trimmed, "/", 2)[0]
	if strings.Contains(hostPort, ":") {
		parts := strings.SplitN(hostPort, ":", 2)
		serverHost = parts[0]
		serverPort = parts[1]
	} else {
		serverHost = hostPort
		if strings.HasPrefix(serverURL, "https://") {
			serverPort = "443"
		} else {
			serverPort = "80"
		}
	}

	// Basic network connectivity test
	if isReachable := utils.TcpPing(serverHost, serverPort); isReachable {
		fmt.Printf("Basic network connectivity: Yes\n")
	} else {
		fmt.Printf("Basic network connectivity: No\n")
	}

	// API credentials and server connectivity test
	if err := pingServer(); err != nil {
		fmt.Printf("❌ Failed\n")
		fmt.Printf("  Error: %v\n", err)
	} else {
		fmt.Printf("Server connectivity and API credentials validated.\n")
	}
	fmt.Printf("\n")

	// Recent Logs
	fmt.Printf("=== Recent Logs (last 10 lines) ===\n")
	if logLines := getRecentLogs(cfg.LogFile); len(logLines) > 0 {
		for _, line := range logLines {
			fmt.Printf("%s\n", line)
		}
	} else {
		fmt.Printf("No recent logs found or log file does not exist\n")
	}

	return nil
}

func getRecentLogs(logFile string) []string {
	var lines []string

	file, err := os.Open(logFile)
	if err != nil {
		return lines
	}
	defer file.Close()

	// Read last 10 lines using tail-like approach
	cmd := exec.Command("tail", "-10", logFile)
	output, err := cmd.Output()
	if err != nil {
		return lines
	}

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	return lines
}
