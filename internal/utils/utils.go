package utils

import (
	"fmt"
	"net"
	"time"
)

// TcpPing performs a simple TCP connection test to the specified host and port
func TcpPing(host, port string) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%s", host, port), 5*time.Second)
	if err != nil {
		return false
	}
	defer func() {
		// Silently ignore close errors for TCP connections in utility function
		// Errors here are extremely rare and non-critical for a connectivity test
		_ = conn.Close()
	}()
	return true
}
