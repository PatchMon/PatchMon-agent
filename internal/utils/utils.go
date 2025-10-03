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
	defer conn.Close()
	return true
}
