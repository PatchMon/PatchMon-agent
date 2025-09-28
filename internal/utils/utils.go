package utils

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"slices"
	"strings"
	"time"
)

func GetKernelVersion() (string, error) {
	if data, err := os.ReadFile("/proc/version"); err == nil {
		fields := slices.Collect(strings.FieldsSeq(string(data)))
		if len(fields) >= 3 {
			return fields[2], nil
		}
	}

	// Fallback to uname
	output, err := exec.Command("uname", "-r").Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}

// TcpPing performs a simple TCP connection test to the specified host and port
func TcpPing(host, port string) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%s", host, port), 5*time.Second)
	if err != nil {
		return false
	}
	defer conn.Close()
	return true
}
