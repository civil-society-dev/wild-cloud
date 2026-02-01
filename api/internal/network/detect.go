package network

import (
	"fmt"
	"net"
	"os/exec"
	"strings"
)

// NetworkInfo holds detected network configuration
type NetworkInfo struct {
	PrimaryIP        string `json:"primary_ip"`
	PrimaryInterface string `json:"primary_interface"`
	Gateway          string `json:"gateway"`
}

// DetectNetworkInfo detects the machine's primary network configuration
// It finds the interface and IP used for internet-bound traffic by checking
// the route to a public DNS server (8.8.8.8)
func DetectNetworkInfo() (*NetworkInfo, error) {
	// Use ip route to find the default route interface and IP
	// This tells us which interface would be used to reach the internet
	cmd := exec.Command("ip", "route", "get", "8.8.8.8")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get network route: %w", err)
	}

	line := string(output)
	// Example output: "8.8.8.8 via 192.168.8.1 dev wlan0 src 192.168.8.152 uid 1000"

	// Extract gateway (after "via")
	gateway := ""
	if idx := strings.Index(line, "via "); idx != -1 {
		rest := line[idx+4:]
		parts := strings.Fields(rest)
		if len(parts) > 0 {
			gateway = parts[0]
		}
	}

	// Extract interface (after "dev")
	iface := ""
	if idx := strings.Index(line, "dev "); idx != -1 {
		rest := line[idx+4:]
		parts := strings.Fields(rest)
		if len(parts) > 0 {
			iface = parts[0]
		}
	}

	// Extract source IP (after "src")
	// This is the IP that would be used as the source for outbound traffic
	ip := ""
	if idx := strings.Index(line, "src "); idx != -1 {
		rest := line[idx+4:]
		parts := strings.Fields(rest)
		if len(parts) > 0 {
			ip = parts[0]
		}
	}

	if ip == "" || iface == "" {
		return nil, fmt.Errorf("could not detect network information from route output: %s", line)
	}

	return &NetworkInfo{
		PrimaryIP:        ip,
		PrimaryInterface: iface,
		Gateway:          gateway,
	}, nil
}

// ResolveDomain resolves a domain name to an IP address using the system DNS resolver
func ResolveDomain(domain string) (string, error) {
	ips, err := net.LookupIP(domain)
	if err != nil {
		return "", fmt.Errorf("failed to resolve domain: %w", err)
	}

	if len(ips) == 0 {
		return "", fmt.Errorf("no IP addresses found for domain")
	}

	// Return the first IPv4 address
	for _, ip := range ips {
		if ipv4 := ip.To4(); ipv4 != nil {
			return ipv4.String(), nil
		}
	}

	// If no IPv4, return first IP
	return ips[0].String(), nil
}
