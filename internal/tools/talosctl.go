package tools

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// Talosctl provides a thin wrapper around the talosctl command-line tool
type Talosctl struct {
	talosconfigPath string
}

// NewTalosctl creates a new Talosctl wrapper
func NewTalosctl() *Talosctl {
	return &Talosctl{}
}

// NewTalosconfigWithConfig creates a new Talosctl wrapper with a specific talosconfig
func NewTalosconfigWithConfig(talosconfigPath string) *Talosctl {
	return &Talosctl{
		talosconfigPath: talosconfigPath,
	}
}

// buildArgs adds talosconfig to args if set
func (t *Talosctl) buildArgs(baseArgs []string) []string {
	if t.talosconfigPath != "" {
		return append([]string{"--talosconfig", t.talosconfigPath}, baseArgs...)
	}
	return baseArgs
}

// GenConfig generates Talos configuration files
func (t *Talosctl) GenConfig(clusterName, endpoint, outputDir string) error {
	args := []string{
		"gen", "config",
		clusterName,
		endpoint,
		"--output-dir", outputDir,
	}

	cmd := exec.Command("talosctl", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("talosctl gen config failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// ApplyConfig applies configuration to a node
func (t *Talosctl) ApplyConfig(nodeIP, configFile string, insecure bool, talosconfigPath string) error {
	args := []string{
		"apply-config",
		"--nodes", nodeIP,
		"--file", configFile,
	}

	if insecure {
		args = append(args, "--insecure")
	}

	cmd := exec.Command("talosctl", args...)
	if talosconfigPath != "" {
		WithTalosconfig(cmd, talosconfigPath)
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("talosctl apply-config failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// DiskInfo represents disk information including path and size
type DiskInfo struct {
	Path string `json:"path"`
	Size int64  `json:"size"`
}

// GetDisks queries available disks from a node (filters to disks > 10GB)
func (t *Talosctl) GetDisks(nodeIP string, insecure bool) ([]DiskInfo, error) {
	args := []string{
		"get", "disks",
		"--nodes", nodeIP,
		"-o", "json",
	}

	if insecure {
		args = append(args, "--insecure")
	}

	// Use jq to slurp the NDJSON into an array (like v.PoC does with jq -s)
	talosCmd := exec.Command("talosctl", args...)
	jqCmd := exec.Command("jq", "-s", ".")

	// Pipe talosctl output to jq
	jqCmd.Stdin, _ = talosCmd.StdoutPipe()

	if err := talosCmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start talosctl: %w", err)
	}

	output, err := jqCmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to process disks JSON: %w\nOutput: %s", err, string(output))
	}

	if err := talosCmd.Wait(); err != nil {
		return nil, fmt.Errorf("talosctl get disks failed: %w", err)
	}

	var result []map[string]interface{}
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse disks JSON: %w", err)
	}

	disks := []DiskInfo{}
	for _, item := range result {
		metadata, ok := item["metadata"].(map[string]interface{})
		if !ok {
			continue
		}

		id, ok := metadata["id"].(string)
		if !ok {
			continue
		}

		spec, ok := item["spec"].(map[string]interface{})
		if !ok {
			continue
		}

		// Extract size - can be float64 or int
		var size int64
		switch v := spec["size"].(type) {
		case float64:
			size = int64(v)
		case int64:
			size = v
		case int:
			size = int64(v)
		default:
			continue
		}

		// Filter to disks > 10GB (like v.PoC does)
		if size > 10000000000 {
			disks = append(disks, DiskInfo{
				Path: "/dev/" + id,
				Size: size,
			})
		}
	}

	return disks, nil
}

// getResourceJSON executes a talosctl get command and returns parsed JSON array
func (t *Talosctl) getResourceJSON(resourceType, nodeIP string, insecure bool) ([]map[string]interface{}, error) {
	args := []string{
		"get", resourceType,
		"--nodes", nodeIP,
		"-o", "json",
	}

	if insecure {
		args = append(args, "--insecure")
	}

	// Use jq to slurp the NDJSON into an array
	talosCmd := exec.Command("talosctl", args...)
	jqCmd := exec.Command("jq", "-s", ".")

	// Pipe talosctl output to jq
	jqCmd.Stdin, _ = talosCmd.StdoutPipe()

	if err := talosCmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start talosctl: %w", err)
	}

	output, err := jqCmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to process %s JSON: %w\nOutput: %s", resourceType, err, string(output))
	}

	if err := talosCmd.Wait(); err != nil {
		return nil, fmt.Errorf("talosctl get %s failed: %w", resourceType, err)
	}

	var result []map[string]interface{}
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse %s JSON: %w", resourceType, err)
	}

	return result, nil
}

// GetLinks queries network interfaces from a node
func (t *Talosctl) GetLinks(nodeIP string, insecure bool) ([]map[string]interface{}, error) {
	return t.getResourceJSON("links", nodeIP, insecure)
}

// GetRoutes queries routing table from a node
func (t *Talosctl) GetRoutes(nodeIP string, insecure bool) ([]map[string]interface{}, error) {
	return t.getResourceJSON("routes", nodeIP, insecure)
}

// GetDefaultInterface finds the interface with the default route
func (t *Talosctl) GetDefaultInterface(nodeIP string, insecure bool) (string, error) {
	routes, err := t.GetRoutes(nodeIP, insecure)
	if err != nil {
		return "", err
	}

	// Find route with destination 0.0.0.0/0 (default route)
	for _, route := range routes {
		if spec, ok := route["spec"].(map[string]interface{}); ok {
			destination, _ := spec["destination"].(string)
			gateway, _ := spec["gateway"].(string)
			if destination == "0.0.0.0/0" && gateway != "" {
				if outLink, ok := spec["outLinkName"].(string); ok {
					return outLink, nil
				}
			}
		}
	}

	return "", fmt.Errorf("no default route found")
}

// GetPhysicalInterface finds the first physical ethernet interface
func (t *Talosctl) GetPhysicalInterface(nodeIP string, insecure bool) (string, error) {
	links, err := t.GetLinks(nodeIP, insecure)
	if err != nil {
		return "", err
	}

	// Look for physical ethernet interfaces (eth*, en*, eno*, ens*, enp*)
	for _, link := range links {
		metadata, ok := link["metadata"].(map[string]interface{})
		if !ok {
			continue
		}

		id, ok := metadata["id"].(string)
		if !ok || id == "lo" {
			continue
		}

		spec, ok := link["spec"].(map[string]interface{})
		if !ok {
			continue
		}

		// Check if it's ethernet and up
		linkType, _ := spec["type"].(string)
		operState, _ := spec["operationalState"].(string)

		if linkType == "ether" && operState == "up" {
			// Prefer interfaces starting with eth, en
			if strings.HasPrefix(id, "eth") || strings.HasPrefix(id, "en") {
				// Skip virtual interfaces (cni, flannel, docker, br-, veth)
				if !strings.Contains(id, "cni") &&
					!strings.Contains(id, "flannel") &&
					!strings.Contains(id, "docker") &&
					!strings.HasPrefix(id, "br-") &&
					!strings.HasPrefix(id, "veth") {
					return id, nil
				}
			}
		}
	}

	return "", fmt.Errorf("no physical ethernet interface found")
}

// GetVersion gets Talos version from a node
func (t *Talosctl) GetVersion(nodeIP string, insecure bool) (string, error) {
	args := t.buildArgs([]string{
		"version",
		"--nodes", nodeIP,
		"--short",
	})

	if insecure {
		args = append(args, "--insecure")
	}

	cmd := exec.Command("talosctl", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("talosctl version failed: %w\nOutput: %s", err, string(output))
	}

	// Parse output to extract server version
	// Output format:
	// Client:
	// Talos v1.11.2
	// Server:
	//   NODE:   ...
	//   Tag:    v1.11.0
	lines := strings.Split(string(output), "\n")
	for i, line := range lines {
		if strings.Contains(line, "Tag:") {
			// Extract version from "Tag: v1.11.0" format
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				return parts[len(parts)-1], nil
			}
		}
		// Also check for simple "Talos vX.Y.Z" format
		if strings.HasPrefix(strings.TrimSpace(line), "Talos v") && i < 3 {
			return strings.TrimSpace(strings.TrimPrefix(line, "Talos ")), nil
		}
	}

	return strings.TrimSpace(string(output)), nil
}

// Validate checks if talosctl is available
func (t *Talosctl) Validate() error {
	cmd := exec.Command("talosctl", "version", "--client")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("talosctl not found or not working: %w\nOutput: %s", err, string(output))
	}
	return nil
}
