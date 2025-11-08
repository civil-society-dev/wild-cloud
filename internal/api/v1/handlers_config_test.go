package v1

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gorilla/mux"
	"gopkg.in/yaml.v3"

	"github.com/wild-cloud/wild-central/daemon/internal/storage"
)

func setupTestAPI(t *testing.T) (*API, string) {
	tmpDir := t.TempDir()
	appsDir := filepath.Join(tmpDir, "apps")

	api, err := NewAPI(tmpDir, appsDir)
	if err != nil {
		t.Fatalf("Failed to create test API: %v", err)
	}

	return api, tmpDir
}

func createTestInstance(t *testing.T, api *API, name string) {
	if err := api.instance.CreateInstance(name); err != nil {
		t.Fatalf("Failed to create test instance: %v", err)
	}
}

func TestUpdateYAMLFile_DeltaUpdate(t *testing.T) {
	api, _ := setupTestAPI(t)
	instanceName := "test-instance"
	createTestInstance(t, api, instanceName)

	configPath := api.instance.GetInstanceConfigPath(instanceName)

	// Create initial config
	initialConfig := map[string]interface{}{
		"domain": "old.com",
		"email":  "admin@old.com",
		"cluster": map[string]interface{}{
			"name": "test-cluster",
		},
	}
	initialYAML, _ := yaml.Marshal(initialConfig)
	if err := storage.WriteFile(configPath, initialYAML, 0644); err != nil {
		t.Fatalf("Failed to write initial config: %v", err)
	}

	// Update only domain
	updateData := map[string]interface{}{
		"domain": "new.com",
	}
	updateYAML, _ := yaml.Marshal(updateData)

	req := httptest.NewRequest("PUT", "/api/v1/instances/"+instanceName+"/config", bytes.NewBuffer(updateYAML))
	w := httptest.NewRecorder()

	vars := map[string]string{"name": instanceName}
	req = mux.SetURLVars(req, vars)

	api.UpdateConfig(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify merged config
	resultData, err := storage.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read result: %v", err)
	}

	var result map[string]interface{}
	if err := yaml.Unmarshal(resultData, &result); err != nil {
		t.Fatalf("Failed to parse result: %v", err)
	}

	// Domain should be updated
	if result["domain"] != "new.com" {
		t.Errorf("Expected domain='new.com', got %v", result["domain"])
	}

	// Email should be preserved
	if result["email"] != "admin@old.com" {
		t.Errorf("Expected email='admin@old.com', got %v", result["email"])
	}

	// Cluster should be preserved
	if cluster, ok := result["cluster"].(map[string]interface{}); !ok {
		t.Errorf("Cluster not preserved as map")
	} else if cluster["name"] != "test-cluster" {
		t.Errorf("Cluster name not preserved")
	}
}

func TestUpdateYAMLFile_FullReplacement(t *testing.T) {
	api, _ := setupTestAPI(t)
	instanceName := "test-instance"
	createTestInstance(t, api, instanceName)

	configPath := api.instance.GetInstanceConfigPath(instanceName)

	// Create initial config
	initialConfig := map[string]interface{}{
		"domain": "old.com",
		"email":  "admin@old.com",
		"oldKey": "oldValue",
	}
	initialYAML, _ := yaml.Marshal(initialConfig)
	if err := storage.WriteFile(configPath, initialYAML, 0644); err != nil {
		t.Fatalf("Failed to write initial config: %v", err)
	}

	// Full replacement
	newConfig := map[string]interface{}{
		"domain": "new.com",
		"email":  "new@new.com",
		"newKey": "newValue",
	}
	newYAML, _ := yaml.Marshal(newConfig)

	req := httptest.NewRequest("PUT", "/api/v1/instances/"+instanceName+"/config", bytes.NewBuffer(newYAML))
	w := httptest.NewRecorder()

	vars := map[string]string{"name": instanceName}
	req = mux.SetURLVars(req, vars)

	api.UpdateConfig(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify result
	resultData, err := storage.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read result: %v", err)
	}

	var result map[string]interface{}
	if err := yaml.Unmarshal(resultData, &result); err != nil {
		t.Fatalf("Failed to parse result: %v", err)
	}

	// All new values should be present
	if result["domain"] != "new.com" {
		t.Errorf("Expected domain='new.com', got %v", result["domain"])
	}
	if result["email"] != "new@new.com" {
		t.Errorf("Expected email='new@new.com', got %v", result["email"])
	}
	if result["newKey"] != "newValue" {
		t.Errorf("Expected newKey='newValue', got %v", result["newKey"])
	}

	// Old key should still be present (shallow merge)
	if result["oldKey"] != "oldValue" {
		t.Errorf("Expected oldKey='oldValue', got %v", result["oldKey"])
	}
}

func TestUpdateYAMLFile_NestedStructure(t *testing.T) {
	api, _ := setupTestAPI(t)
	instanceName := "test-instance"
	createTestInstance(t, api, instanceName)

	configPath := api.instance.GetInstanceConfigPath(instanceName)

	// Update with nested structure
	updateData := map[string]interface{}{
		"cloud": map[string]interface{}{
			"domain": "test.com",
			"dns": map[string]interface{}{
				"ip": "1.2.3.4",
				"port": 53,
			},
		},
	}
	updateYAML, _ := yaml.Marshal(updateData)

	req := httptest.NewRequest("PUT", "/api/v1/instances/"+instanceName+"/config", bytes.NewBuffer(updateYAML))
	w := httptest.NewRecorder()

	vars := map[string]string{"name": instanceName}
	req = mux.SetURLVars(req, vars)

	api.UpdateConfig(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify nested structure preserved
	resultData, err := storage.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read result: %v", err)
	}

	var result map[string]interface{}
	if err := yaml.Unmarshal(resultData, &result); err != nil {
		t.Fatalf("Failed to parse result: %v", err)
	}

	// Verify nested structure is proper YAML, not Go map notation
	resultStr := string(resultData)
	if bytes.Contains(resultData, []byte("map[")) {
		t.Errorf("Result contains Go map notation: %s", resultStr)
	}

	// Verify structure is accessible
	cloud, ok := result["cloud"].(map[string]interface{})
	if !ok {
		t.Fatalf("cloud is not a map: %T", result["cloud"])
	}

	if cloud["domain"] != "test.com" {
		t.Errorf("Expected cloud.domain='test.com', got %v", cloud["domain"])
	}

	dns, ok := cloud["dns"].(map[string]interface{})
	if !ok {
		t.Fatalf("cloud.dns is not a map: %T", cloud["dns"])
	}

	if dns["ip"] != "1.2.3.4" {
		t.Errorf("Expected dns.ip='1.2.3.4', got %v", dns["ip"])
	}
	if dns["port"] != 53 {
		t.Errorf("Expected dns.port=53, got %v", dns["port"])
	}
}

func TestUpdateYAMLFile_EmptyFileCreation(t *testing.T) {
	api, _ := setupTestAPI(t)
	instanceName := "test-instance"
	createTestInstance(t, api, instanceName)

	configPath := api.instance.GetInstanceConfigPath(instanceName)

	// Truncate the config file to make it empty (but still exists)
	if err := storage.WriteFile(configPath, []byte(""), 0644); err != nil {
		t.Fatalf("Failed to empty config file: %v", err)
	}

	// Update should populate empty file
	updateData := map[string]interface{}{
		"domain": "new.com",
		"email":  "admin@new.com",
	}
	updateYAML, _ := yaml.Marshal(updateData)

	req := httptest.NewRequest("PUT", "/api/v1/instances/"+instanceName+"/config", bytes.NewBuffer(updateYAML))
	w := httptest.NewRecorder()

	vars := map[string]string{"name": instanceName}
	req = mux.SetURLVars(req, vars)

	api.UpdateConfig(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify content
	resultData, err := storage.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read result: %v", err)
	}

	var result map[string]interface{}
	if err := yaml.Unmarshal(resultData, &result); err != nil {
		t.Fatalf("Failed to parse result: %v", err)
	}

	if result["domain"] != "new.com" {
		t.Errorf("Expected domain='new.com', got %v", result["domain"])
	}
	if result["email"] != "admin@new.com" {
		t.Errorf("Expected email='admin@new.com', got %v", result["email"])
	}
}

func TestUpdateYAMLFile_EmptyUpdate(t *testing.T) {
	api, _ := setupTestAPI(t)
	instanceName := "test-instance"
	createTestInstance(t, api, instanceName)

	configPath := api.instance.GetInstanceConfigPath(instanceName)

	// Create initial config
	initialConfig := map[string]interface{}{
		"domain": "test.com",
	}
	initialYAML, _ := yaml.Marshal(initialConfig)
	if err := storage.WriteFile(configPath, initialYAML, 0644); err != nil {
		t.Fatalf("Failed to write initial config: %v", err)
	}

	// Empty update
	updateData := map[string]interface{}{}
	updateYAML, _ := yaml.Marshal(updateData)

	req := httptest.NewRequest("PUT", "/api/v1/instances/"+instanceName+"/config", bytes.NewBuffer(updateYAML))
	w := httptest.NewRecorder()

	vars := map[string]string{"name": instanceName}
	req = mux.SetURLVars(req, vars)

	api.UpdateConfig(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify file unchanged
	resultData, err := storage.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read result: %v", err)
	}

	var result map[string]interface{}
	if err := yaml.Unmarshal(resultData, &result); err != nil {
		t.Fatalf("Failed to parse result: %v", err)
	}

	if result["domain"] != "test.com" {
		t.Errorf("Expected domain='test.com', got %v", result["domain"])
	}
}

func TestUpdateYAMLFile_YAMLFormatting(t *testing.T) {
	api, _ := setupTestAPI(t)
	instanceName := "test-instance"
	createTestInstance(t, api, instanceName)

	configPath := api.instance.GetInstanceConfigPath(instanceName)

	// Update with complex nested structure
	updateData := map[string]interface{}{
		"cloud": map[string]interface{}{
			"domain": "test.com",
			"dns": map[string]interface{}{
				"ip": "1.2.3.4",
			},
		},
		"cluster": map[string]interface{}{
			"nodes": []interface{}{
				map[string]interface{}{
					"name": "node1",
					"ip":   "10.0.0.1",
				},
				map[string]interface{}{
					"name": "node2",
					"ip":   "10.0.0.2",
				},
			},
		},
	}
	updateYAML, _ := yaml.Marshal(updateData)

	req := httptest.NewRequest("PUT", "/api/v1/instances/"+instanceName+"/config", bytes.NewBuffer(updateYAML))
	w := httptest.NewRecorder()

	vars := map[string]string{"name": instanceName}
	req = mux.SetURLVars(req, vars)

	api.UpdateConfig(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify YAML formatting
	resultData, err := storage.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read result: %v", err)
	}

	resultStr := string(resultData)

	// Should not contain Go map notation
	if bytes.Contains(resultData, []byte("map[")) {
		t.Errorf("Result contains Go map notation: %s", resultStr)
	}

	// Should be valid YAML
	var result map[string]interface{}
	if err := yaml.Unmarshal(resultData, &result); err != nil {
		t.Fatalf("Result is not valid YAML: %v", err)
	}

	// Should have proper indentation (check for nested structure indicators)
	if !bytes.Contains(resultData, []byte("  ")) {
		t.Error("Result appears to lack proper indentation")
	}
}

func TestUpdateYAMLFile_InvalidYAML(t *testing.T) {
	api, _ := setupTestAPI(t)
	instanceName := "test-instance"
	createTestInstance(t, api, instanceName)

	// Send invalid YAML
	invalidYAML := []byte("invalid: yaml: content: [")

	req := httptest.NewRequest("PUT", "/api/v1/instances/"+instanceName+"/config", bytes.NewBuffer(invalidYAML))
	w := httptest.NewRecorder()

	vars := map[string]string{"name": instanceName}
	req = mux.SetURLVars(req, vars)

	api.UpdateConfig(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestUpdateYAMLFile_InvalidInstance(t *testing.T) {
	api, _ := setupTestAPI(t)

	updateData := map[string]interface{}{
		"domain": "test.com",
	}
	updateYAML, _ := yaml.Marshal(updateData)

	req := httptest.NewRequest("PUT", "/api/v1/instances/nonexistent/config", bytes.NewBuffer(updateYAML))
	w := httptest.NewRecorder()

	vars := map[string]string{"name": "nonexistent"}
	req = mux.SetURLVars(req, vars)

	api.UpdateConfig(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestUpdateYAMLFile_FilePermissions(t *testing.T) {
	api, _ := setupTestAPI(t)
	instanceName := "test-instance"
	createTestInstance(t, api, instanceName)

	configPath := api.instance.GetInstanceConfigPath(instanceName)

	updateData := map[string]interface{}{
		"domain": "test.com",
	}
	updateYAML, _ := yaml.Marshal(updateData)

	req := httptest.NewRequest("PUT", "/api/v1/instances/"+instanceName+"/config", bytes.NewBuffer(updateYAML))
	w := httptest.NewRecorder()

	vars := map[string]string{"name": instanceName}
	req = mux.SetURLVars(req, vars)

	api.UpdateConfig(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Check file permissions
	info, err := os.Stat(configPath)
	if err != nil {
		t.Fatalf("Failed to stat config file: %v", err)
	}

	expectedPerm := os.FileMode(0644)
	if info.Mode().Perm() != expectedPerm {
		t.Errorf("Expected permissions %v, got %v", expectedPerm, info.Mode().Perm())
	}
}

func TestUpdateYAMLFile_UpdateSecrets(t *testing.T) {
	api, _ := setupTestAPI(t)
	instanceName := "test-instance"
	createTestInstance(t, api, instanceName)

	secretsPath := api.instance.GetInstanceSecretsPath(instanceName)

	// Update secrets
	updateData := map[string]interface{}{
		"dbPassword":  "secret123",
		"apiKey":      "key456",
	}
	updateYAML, _ := yaml.Marshal(updateData)

	req := httptest.NewRequest("PUT", "/api/v1/instances/"+instanceName+"/secrets", bytes.NewBuffer(updateYAML))
	w := httptest.NewRecorder()

	vars := map[string]string{"name": instanceName}
	req = mux.SetURLVars(req, vars)

	api.UpdateSecrets(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify secrets file created and contains data
	resultData, err := storage.ReadFile(secretsPath)
	if err != nil {
		t.Fatalf("Failed to read secrets: %v", err)
	}

	var result map[string]interface{}
	if err := yaml.Unmarshal(resultData, &result); err != nil {
		t.Fatalf("Failed to parse secrets: %v", err)
	}

	if result["dbPassword"] != "secret123" {
		t.Errorf("Expected dbPassword='secret123', got %v", result["dbPassword"])
	}
	if result["apiKey"] != "key456" {
		t.Errorf("Expected apiKey='key456', got %v", result["apiKey"])
	}
}

func TestUpdateYAMLFile_ConcurrentUpdates(t *testing.T) {
	api, _ := setupTestAPI(t)
	instanceName := "test-instance"
	createTestInstance(t, api, instanceName)

	// This test verifies that file locking prevents race conditions
	// We'll simulate concurrent updates and verify data integrity

	numUpdates := 10
	done := make(chan bool, numUpdates)

	for i := 0; i < numUpdates; i++ {
		go func(index int) {
			updateData := map[string]interface{}{
				"counter": index,
			}
			updateYAML, _ := yaml.Marshal(updateData)

			req := httptest.NewRequest("PUT", "/api/v1/instances/"+instanceName+"/config", bytes.NewBuffer(updateYAML))
			w := httptest.NewRecorder()

			vars := map[string]string{"name": instanceName}
			req = mux.SetURLVars(req, vars)

			api.UpdateConfig(w, req)

			done <- w.Code == http.StatusOK
		}(i)
	}

	// Wait for all updates to complete
	successCount := 0
	for i := 0; i < numUpdates; i++ {
		if <-done {
			successCount++
		}
	}

	if successCount != numUpdates {
		t.Errorf("Expected %d successful updates, got %d", numUpdates, successCount)
	}

	// Verify file is still valid YAML
	configPath := api.instance.GetInstanceConfigPath(instanceName)
	resultData, err := storage.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read final config: %v", err)
	}

	var result map[string]interface{}
	if err := yaml.Unmarshal(resultData, &result); err != nil {
		t.Fatalf("Final config is not valid YAML: %v", err)
	}
}

func TestUpdateYAMLFile_PreservesComplexTypes(t *testing.T) {
	api, _ := setupTestAPI(t)
	instanceName := "test-instance"
	createTestInstance(t, api, instanceName)

	configPath := api.instance.GetInstanceConfigPath(instanceName)

	// Create config with various types
	updateData := map[string]interface{}{
		"stringValue": "text",
		"intValue":    42,
		"floatValue":  3.14,
		"boolValue":   true,
		"arrayValue":  []interface{}{"a", "b", "c"},
		"mapValue": map[string]interface{}{
			"nested": "value",
		},
		"nullValue": nil,
	}
	updateYAML, _ := yaml.Marshal(updateData)

	req := httptest.NewRequest("PUT", "/api/v1/instances/"+instanceName+"/config", bytes.NewBuffer(updateYAML))
	w := httptest.NewRecorder()

	vars := map[string]string{"name": instanceName}
	req = mux.SetURLVars(req, vars)

	api.UpdateConfig(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify types preserved
	resultData, err := storage.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read result: %v", err)
	}

	var result map[string]interface{}
	if err := yaml.Unmarshal(resultData, &result); err != nil {
		t.Fatalf("Failed to parse result: %v", err)
	}

	if result["stringValue"] != "text" {
		t.Errorf("String value not preserved: %v", result["stringValue"])
	}
	if result["intValue"] != 42 {
		t.Errorf("Int value not preserved: %v", result["intValue"])
	}
	if result["floatValue"] != 3.14 {
		t.Errorf("Float value not preserved: %v", result["floatValue"])
	}
	if result["boolValue"] != true {
		t.Errorf("Bool value not preserved: %v", result["boolValue"])
	}

	arrayValue, ok := result["arrayValue"].([]interface{})
	if !ok {
		t.Errorf("Array not preserved as slice: %T", result["arrayValue"])
	} else if len(arrayValue) != 3 {
		t.Errorf("Array length not preserved: %d", len(arrayValue))
	}

	mapValue, ok := result["mapValue"].(map[string]interface{})
	if !ok {
		t.Errorf("Map not preserved: %T", result["mapValue"])
	} else if mapValue["nested"] != "value" {
		t.Errorf("Nested map value not preserved: %v", mapValue["nested"])
	}

	if result["nullValue"] != nil {
		t.Errorf("Null value not preserved: %v", result["nullValue"])
	}
}
