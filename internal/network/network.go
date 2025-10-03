package network

import (
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/sirupsen/logrus"

	"patchmon-agent/internal/constants"
	"patchmon-agent/pkg/models"
)

// Manager handles network information collection using standard library and file parsing
type Manager struct {
	logger *logrus.Logger
}

// New creates a new network manager
func New(logger *logrus.Logger) *Manager {
	return &Manager{
		logger: logger,
	}
}

// GetNetworkInfo collects network information
func (m *Manager) GetNetworkInfo() models.NetworkInfo {
	m.logger.Debug("Collecting network information...")

	info := models.NetworkInfo{
		GatewayIP:         m.getGatewayIP(),
		DNSServers:        m.getDNSServers(),
		NetworkInterfaces: m.getNetworkInterfaces(),
	}

	m.logger.WithFields(logrus.Fields{
		"gateway":    info.GatewayIP,
		"dns":        len(info.DNSServers),
		"interfaces": len(info.NetworkInterfaces),
	}).Debug("Network info collected")

	return info
}

// getGatewayIP gets the default gateway IP from routing table file
func (m *Manager) getGatewayIP() string {
	// Read /proc/net/route to find default gateway
	data, err := os.ReadFile("/proc/net/route")
	if err != nil {
		m.logger.WithError(err).Warn("Failed to read /proc/net/route")
		return ""
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 3 && fields[1] == "00000000" { // Default route
			// Convert hex gateway to IP
			if gateway := m.hexToIP(fields[2]); gateway != "" {
				return gateway
			}
		}
	}

	return ""
}

// hexToIP converts hex IP address to dotted decimal notation
func (m *Manager) hexToIP(hexIP string) string {
	if len(hexIP) != 8 {
		return ""
	}

	// Convert little-endian hex to IP
	ip := make([]byte, 4)
	for i := 0; i < 4; i++ {
		if val, err := parseHexByte(hexIP[6-i*2 : 8-i*2]); err == nil {
			ip[i] = val
		} else {
			return ""
		}
	}

	return net.IP(ip).String()
}

// parseHexByte parses a 2-character hex string to byte
func parseHexByte(hex string) (byte, error) {
	var result byte
	for _, c := range hex {
		result <<= 4
		if c >= '0' && c <= '9' {
			result += byte(c - '0')
		} else if c >= 'A' && c <= 'F' {
			result += byte(c - 'A' + 10)
		} else if c >= 'a' && c <= 'f' {
			result += byte(c - 'a' + 10)
		} else {
			return 0, fmt.Errorf("invalid hex character: %c", c)
		}
	}
	return result, nil
}

// getDNSServers gets the configured DNS servers from resolv.conf
func (m *Manager) getDNSServers() []string {
	var servers []string

	// Read /etc/resolv.conf
	data, err := os.ReadFile("/etc/resolv.conf")
	if err != nil {
		m.logger.WithError(err).Warn("Failed to read /etc/resolv.conf")
		return servers
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "nameserver") {
			fields := strings.Fields(line)
			if len(fields) > 1 {
				servers = append(servers, fields[1])
			}
		}
	}

	return servers
}

// getNetworkInterfaces gets network interface information using standard library
func (m *Manager) getNetworkInterfaces() []models.NetworkInterface {
	interfaces, err := net.Interfaces()
	if err != nil {
		m.logger.WithError(err).Warn("Failed to get network interfaces")
		return []models.NetworkInterface{}
	}

	var result []models.NetworkInterface

	for _, iface := range interfaces {
		// Skip loopback interface
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		// Get IP addresses for this interface
		var addresses []models.NetworkAddress

		addrs, err := iface.Addrs()
		if err != nil {
			m.logger.WithError(err).WithField("interface", iface.Name).Warn("Failed to get addresses for interface")
			continue
		}

		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok {
				var family string
				if ipnet.IP.To4() != nil {
					family = constants.IPFamilyIPv4
				} else {
					family = constants.IPFamilyIPv6
				}

				addresses = append(addresses, models.NetworkAddress{
					Address: ipnet.IP.String(),
					Family:  family,
				})
			}
		}

		// Only include interfaces that have addresses
		if len(addresses) > 0 {
			// Determine interface type
			interfaceType := constants.NetTypeEthernet
			if strings.HasPrefix(iface.Name, "wl") || strings.HasPrefix(iface.Name, "wifi") {
				interfaceType = constants.NetTypeWiFi
			} else if strings.HasPrefix(iface.Name, "docker") || strings.HasPrefix(iface.Name, "br-") {
				interfaceType = constants.NetTypeBridge
			}

			result = append(result, models.NetworkInterface{
				Name:      iface.Name,
				Type:      interfaceType,
				Addresses: addresses,
			})
		}
	}

	return result
}
