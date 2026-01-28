package v1

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wild-cloud/wild-central/daemon/internal/config"
	"github.com/wild-cloud/wild-central/daemon/internal/storage"
	"gopkg.in/yaml.v3"
)

func setupTestDnsmasq(t *testing.T) (*API, string) {
	tmpDir := t.TempDir()

	// Set dnsmasq config to temp directory
	dnsmasqConfigPath := filepath.Join(tmpDir, "dnsmasq.conf")
	os.Setenv("WILD_API_DNSMASQ_CONFIG_PATH", dnsmasqConfigPath)
	t.Cleanup(func() {
		os.Unsetenv("WILD_API_DNSMASQ_CONFIG_PATH")
	})

	appsDir := filepath.Join(tmpDir, "apps")
	api, err := NewAPI(tmpDir, appsDir)
	if err != nil {
		t.Fatalf("Failed to create test API: %v", err)
	}

	return api, tmpDir
}

func TestDnsmasqGenerate_WithoutOverwrite(t *testing.T) {
	api, tmpDir := setupTestDnsmasq(t)

	// Create global config
	globalConfig := config.GlobalConfig{}
	globalConfig.Cloud.DNS.IP = "192.168.1.100"
	globalConfig.Cloud.Dnsmasq.Interface = "eth0"
	configPath := filepath.Join(tmpDir, "config.yaml")
	configData, _ := yaml.Marshal(globalConfig)
	storage.WriteFile(configPath, configData, 0644)

	// Create test instance
	instanceName := "test-instance"
	createTestInstance(t, api, instanceName)

	// Create instance config
	instanceConfig := config.InstanceConfig{}
	instanceConfig.Cloud.Domain = "test.local"
	instanceConfig.Cloud.InternalDomain = "internal.test.local"
	instanceConfigPath := api.instance.GetInstanceConfigPath(instanceName)
	instanceConfigData, _ := yaml.Marshal(instanceConfig)
	storage.WriteFile(instanceConfigPath, instanceConfigData, 0644)

	// Test generate without overwrite
	req := httptest.NewRequest("POST", "/api/v1/dnsmasq/generate", nil)
	w := httptest.NewRecorder()

	api.DnsmasqGenerate(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	// Verify response contains config
	if config, ok := resp["config"].(string); !ok || config == "" {
		t.Fatal("Expected generated config in response")
	}

	// Verify config was NOT written to file
	configPath = api.dnsmasq.GetConfigPath()
	if _, err := os.Stat(configPath); err == nil {
		// File should not exist yet
		t.Fatal("Config file should not exist in dry-run mode")
	}
}

func TestDnsmasqGenerate_WithOverwrite(t *testing.T) {
	api, tmpDir := setupTestDnsmasq(t)

	// Create global config
	globalConfig := config.GlobalConfig{}
	globalConfig.Cloud.DNS.IP = "192.168.1.100"
	globalConfig.Cloud.Dnsmasq.Interface = "eth0"
	configPath := filepath.Join(tmpDir, "config.yaml")
	configData, _ := yaml.Marshal(globalConfig)
	storage.WriteFile(configPath, configData, 0644)

	// Create test instance
	instanceName := "test-instance"
	createTestInstance(t, api, instanceName)

	// Create instance config
	instanceConfig := config.InstanceConfig{}
	instanceConfig.Cloud.Domain = "test.local"
	instanceConfig.Cloud.InternalDomain = "internal.test.local"
	instanceConfig.Cluster.LoadBalancerIp = "192.168.1.80"
	instanceConfigPath := api.instance.GetInstanceConfigPath(instanceName)
	instanceConfigData, _ := yaml.Marshal(instanceConfig)
	storage.WriteFile(instanceConfigPath, instanceConfigData, 0644)

	// Instead of calling the handler which would try to restart the service,
	// directly test the UpdateConfig method with restart=false
	instances, _ := api.instance.ListInstances()
	var instanceConfigs []config.InstanceConfig
	for _, name := range instances {
		icPath := api.instance.GetInstanceConfigPath(name)
		ic, _ := config.LoadCloudConfig(icPath)
		instanceConfigs = append(instanceConfigs, *ic)
	}

	// Test UpdateConfig with restart=false (safe for tests)
	err := api.dnsmasq.UpdateConfig(&globalConfig, instanceConfigs, false)
	if err != nil {
		t.Fatalf("UpdateConfig failed: %v", err)
	}

	// Verify config was written
	dnsmasqConfigPath := api.dnsmasq.GetConfigPath()
	content, err := os.ReadFile(dnsmasqConfigPath)
	if err != nil {
		t.Fatalf("Failed to read dnsmasq config: %v", err)
	}

	configStr := string(content)
	if !strings.Contains(configStr, "test.local") {
		t.Fatal("Expected config to contain test.local domain")
	}
	if !strings.Contains(configStr, "internal.test.local") {
		t.Fatal("Expected config to contain internal.test.local domain")
	}
	// Check for load balancer IP in resolution section (DNS IP is auto-detected from network)
	if !strings.Contains(configStr, "192.168.1.80") {
		t.Fatal("Expected config to contain load balancer IP")
	}
}

func TestDnsmasqWriteConfig(t *testing.T) {
	api, _ := setupTestDnsmasq(t)

	customConfig := `# Custom dnsmasq config
interface=wlan0
listen-address=192.168.2.1
`

	reqBody := map[string]string{
		"content": customConfig,
	}
	reqData, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("PUT", "/api/v1/dnsmasq/config", bytes.NewBuffer(reqData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	api.DnsmasqWriteConfig(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify config was written
	configPath := api.dnsmasq.GetConfigPath()
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read written config: %v", err)
	}

	if string(content) != customConfig {
		t.Fatalf("Config content mismatch.\nExpected:\n%s\nGot:\n%s", customConfig, string(content))
	}
}

func TestDnsmasqWriteConfig_EmptyContent(t *testing.T) {
	api, _ := setupTestDnsmasq(t)

	reqBody := map[string]string{
		"content": "",
	}
	reqData, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("PUT", "/api/v1/dnsmasq/config", bytes.NewBuffer(reqData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	api.DnsmasqWriteConfig(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("Expected status 400, got %d", w.Code)
	}
}

func TestDnsmasqGetConfig(t *testing.T) {
	api, _ := setupTestDnsmasq(t)

	// Write a config first
	configPath := api.dnsmasq.GetConfigPath()
	testConfig := "# Test config\ninterface=eth0\n"
	os.MkdirAll(filepath.Dir(configPath), 0755)
	os.WriteFile(configPath, []byte(testConfig), 0644)

	req := httptest.NewRequest("GET", "/api/v1/dnsmasq/config", nil)
	w := httptest.NewRecorder()

	api.DnsmasqGetConfig(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	content, ok := resp["content"].(string)
	if !ok || content != testConfig {
		t.Fatalf("Expected config content: %s, got: %s", testConfig, content)
	}
}
