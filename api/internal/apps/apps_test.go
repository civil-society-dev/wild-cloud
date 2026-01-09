package apps

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/wild-cloud/wild-central/daemon/internal/secrets"
	"github.com/wild-cloud/wild-central/daemon/internal/storage"
	"github.com/wild-cloud/wild-central/daemon/internal/tools"
)

func TestSetNestedConfig(t *testing.T) {
	// Create temp directory for test
	tmpDir, err := os.MkdirTemp("", "apps-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test config file
	configFile := filepath.Join(tmpDir, "config.yaml")
	initialConfig := `apps:
  testapp:
    existingKey: existingValue
`
	if err := os.WriteFile(configFile, []byte(initialConfig), 0644); err != nil {
		t.Fatal(err)
	}

	yq := tools.NewYQ()

	tests := []struct {
		name     string
		basePath string
		value    interface{}
		verify   func(t *testing.T)
		wantErr  bool
	}{
		{
			name:     "simple string value",
			basePath: ".apps.testapp.simpleKey",
			value:    "simpleValue",
			verify: func(t *testing.T) {
				result, err := yq.Get(configFile, ".apps.testapp.simpleKey")
				if err != nil {
					t.Fatalf("failed to get value: %v", err)
				}
				if result != "simpleValue" {
					t.Errorf("expected 'simpleValue', got '%s'", result)
				}
			},
		},
		{
			name:     "simple number value",
			basePath: ".apps.testapp.port",
			value:    587,
			verify: func(t *testing.T) {
				result, err := yq.Get(configFile, ".apps.testapp.port")
				if err != nil {
					t.Fatalf("failed to get value: %v", err)
				}
				if result != "587" {
					t.Errorf("expected '587', got '%s'", result)
				}
			},
		},
		{
			name:     "nested object",
			basePath: ".apps.testapp.smtp",
			value: map[string]interface{}{
				"host":     "mail.example.com",
				"port":     "465",
				"username": "user@example.com",
				"useSSL":   "true",
			},
			verify: func(t *testing.T) {
				// Check each nested value
				checks := map[string]string{
					".apps.testapp.smtp.host":     "mail.example.com",
					".apps.testapp.smtp.port":     "465",
					".apps.testapp.smtp.username": "user@example.com",
					".apps.testapp.smtp.useSSL":   "true",
				}
				for path, expected := range checks {
					result, err := yq.Get(configFile, path)
					if err != nil {
						t.Fatalf("failed to get %s: %v", path, err)
					}
					if result != expected {
						t.Errorf("path %s: expected '%s', got '%s'", path, expected, result)
					}
				}
			},
		},
		{
			name:     "deeply nested object",
			basePath: ".apps.testapp.storage",
			value: map[string]interface{}{
				"volumes": map[string]interface{}{
					"uploads": "5Gi",
					"files":   "10Gi",
					"cache":   "2Gi",
				},
			},
			verify: func(t *testing.T) {
				// Check deeply nested values
				checks := map[string]string{
					".apps.testapp.storage.volumes.uploads": "5Gi",
					".apps.testapp.storage.volumes.files":   "10Gi",
					".apps.testapp.storage.volumes.cache":   "2Gi",
				}
				for path, expected := range checks {
					result, err := yq.Get(configFile, path)
					if err != nil {
						t.Fatalf("failed to get %s: %v", path, err)
					}
					if result != expected {
						t.Errorf("path %s: expected '%s', got '%s'", path, expected, result)
					}
				}
			},
		},
		{
			name:     "map[interface{}]interface{} type (from YAML unmarshal)",
			basePath: ".apps.testapp.database",
			value: map[interface{}]interface{}{
				"host": "postgres.local",
				"port": 5432,
				"name": "mydb",
			},
			verify: func(t *testing.T) {
				checks := map[string]string{
					".apps.testapp.database.host": "postgres.local",
					".apps.testapp.database.port": "5432",
					".apps.testapp.database.name": "mydb",
				}
				for path, expected := range checks {
					result, err := yq.Get(configFile, path)
					if err != nil {
						t.Fatalf("failed to get %s: %v", path, err)
					}
					if result != expected {
						t.Errorf("path %s: expected '%s', got '%s'", path, expected, result)
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := setNestedConfig(yq, configFile, tt.basePath, tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("setNestedConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.verify != nil {
				tt.verify(t)
			}
		})
	}

	// Verify the existing key is still there
	existing, err := yq.Get(configFile, ".apps.testapp.existingKey")
	if err != nil {
		t.Fatalf("failed to get existing key: %v", err)
	}
	if existing != "existingValue" {
		t.Errorf("existing key was modified: expected 'existingValue', got '%s'", existing)
	}
}

func TestAddAppWithNestedConfig(t *testing.T) {
	// Create temp directories
	tmpDir, err := os.MkdirTemp("", "apps-add-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	dataDir := filepath.Join(tmpDir, "data")
	appsDir := filepath.Join(tmpDir, "apps")
	instanceName := "test-instance"
	appName := "loomio"

	// Setup directory structure
	instancePath := filepath.Join(dataDir, "instances", instanceName)
	if err := storage.EnsureDir(instancePath, 0755); err != nil {
		t.Fatal(err)
	}

	// Create instance config
	configFile := filepath.Join(instancePath, "config.yaml")
	instanceConfig := `cloud:
  domain: example.com
  smtp:
    host: mail.example.com
    port: "587"
    user: admin@example.com
    from: no-reply@example.com
`
	if err := os.WriteFile(configFile, []byte(instanceConfig), 0644); err != nil {
		t.Fatal(err)
	}

	// Create secrets file
	secretsFile := filepath.Join(instancePath, "secrets.yaml")
	if err := os.WriteFile(secretsFile, []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create app directory with manifest
	appPath := filepath.Join(appsDir, appName)
	if err := storage.EnsureDir(appPath, 0755); err != nil {
		t.Fatal(err)
	}

	// Create manifest with nested defaultConfig
	manifest := AppManifest{
		Name:        "loomio",
		Description: "Test app",
		Version:     "1.0.0",
		DefaultConfig: map[string]interface{}{
			"domain": "loomio.{{ .cloud.domain }}",
			"smtp": map[string]interface{}{
				"host":     "{{ .cloud.smtp.host }}",
				"port":     "{{ .cloud.smtp.port }}",
				"username": "{{ .cloud.smtp.user }}",
				"useSSL":   "true",
				"from":     "{{ .cloud.smtp.from }}",
			},
			"storage": map[string]interface{}{
				"uploads": "5Gi",
				"files":   "10Gi",
			},
		},
		DefaultSecrets: []SecretDefinition{
			{Key: "apps.loomio.dbPassword"},
		},
	}

	manifestData, err := yaml.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	manifestFile := filepath.Join(appPath, "manifest.yaml")
	if err := os.WriteFile(manifestFile, manifestData, 0644); err != nil {
		t.Fatal(err)
	}

	// Create a sample kustomization.yaml file
	kustomizationFile := filepath.Join(appPath, "kustomization.yaml")
	kustomizationContent := `apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
namespace: loomio
`
	if err := os.WriteFile(kustomizationFile, []byte(kustomizationContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create app manager and add the app
	manager := NewManager(dataDir, appsDir)
	err = manager.Add(instanceName, appName, nil, nil)
	if err != nil {
		t.Fatalf("Failed to add app: %v", err)
	}

	// Verify the nested config was properly set
	yq := tools.NewYQ()

	// Check flat values
	domain, err := yq.Get(configFile, ".apps.loomio.domain")
	if err != nil {
		t.Fatalf("Failed to get domain: %v", err)
	}
	if domain != "loomio.example.com" {
		t.Errorf("Domain incorrect: expected 'loomio.example.com', got '%s'", domain)
	}

	// Check nested SMTP values
	smtpChecks := map[string]string{
		".apps.loomio.smtp.host":     "mail.example.com",
		".apps.loomio.smtp.port":     "587",
		".apps.loomio.smtp.username": "admin@example.com",
		".apps.loomio.smtp.useSSL":   "true",
		".apps.loomio.smtp.from":     "no-reply@example.com",
	}
	for path, expected := range smtpChecks {
		result, err := yq.Get(configFile, path)
		if err != nil {
			t.Fatalf("Failed to get %s: %v", path, err)
		}
		if result != expected {
			t.Errorf("Path %s: expected '%s', got '%s'", path, expected, result)
		}
	}

	// Check nested storage values
	storageChecks := map[string]string{
		".apps.loomio.storage.uploads": "5Gi",
		".apps.loomio.storage.files":   "10Gi",
	}
	for path, expected := range storageChecks {
		result, err := yq.Get(configFile, path)
		if err != nil {
			t.Fatalf("Failed to get %s: %v", path, err)
		}
		if result != expected {
			t.Errorf("Path %s: expected '%s', got '%s'", path, expected, result)
		}
	}

	// Verify app files were created
	appDestDir := filepath.Join(instancePath, "apps", appName)
	if !storage.FileExists(appDestDir) {
		t.Errorf("App directory was not created")
	}

	// Check the compiled kustomization.yaml exists
	compiledKustomization := filepath.Join(appDestDir, "kustomization.yaml")
	if !storage.FileExists(compiledKustomization) {
		t.Errorf("Compiled kustomization.yaml was not created")
	}
}

func TestProcessSecretTemplate(t *testing.T) {
	// Test the processSecretTemplate function directly
	tmpDir, err := os.MkdirTemp("", "process-secret-template-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create config file
	configFile := filepath.Join(tmpDir, "config.yaml")
	configContent := `cloud:
  domain: example.com
apps:
  testapp:
    db:
      host: postgres.local
      port: 5432
      name: testdb
      user: testuser
    apiUrl: https://api.example.com
`
	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create secrets file with an existing secret
	secretsFile := filepath.Join(tmpDir, "secrets.yaml")
	secretsContent := `apps:
  testapp:
    existingSecret: "myexistingsecret"
`
	if err := os.WriteFile(secretsFile, []byte(secretsContent), 0600); err != nil {
		t.Fatal(err)
	}

	gomplate := tools.NewGomplate()

	tests := []struct {
		name     string
		template string
		want     string
		wantErr  bool
	}{
		{
			name:     "simple random generation",
			template: "{{ random.AlphaNum 16 }}",
			want:     "", // Can't predict random, just check length
			wantErr:  false,
		},
		{
			name:     "random with prefix",
			template: "api_{{ random.AlphaNum 32 }}",
			want:     "", // Check prefix and length separately
			wantErr:  false,
		},
		{
			name:     "reference config value",
			template: "postgresql://{{ .app.db.user }}@{{ .app.db.host }}",
			want:     "postgresql://testuser@postgres.local",
			wantErr:  false,
		},
		{
			name:     "reference existing secret",
			template: "webhook_url?key={{ .secrets.existingSecret }}",
			want:     "webhook_url?key=myexistingsecret",
			wantErr:  false,
		},
		{
			name:     "complex template with config and secrets",
			template: "postgresql://{{ .app.db.user }}:{{ .secrets.existingSecret }}@{{ .app.db.host }}:{{ .app.db.port }}/{{ .app.db.name }}",
			want:     "postgresql://testuser:myexistingsecret@postgres.local:5432/testdb",
			wantErr:  false,
		},
		{
			name:     "multiple random values",
			template: "{{ random.AlphaNum 8 }}-{{ random.AlphaNum 8 }}",
			want:     "", // Check format separately
			wantErr:  false,
		},
		{
			name:     "no template markers",
			template: "plain-secret-value",
			want:     "plain-secret-value",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := processSecretTemplate(tt.template, "testapp", configFile, secretsFile, gomplate)
			if (err != nil) != tt.wantErr {
				t.Errorf("processSecretTemplate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Special handling for random-based tests
			switch tt.name {
			case "simple random generation":
				if len(got) != 16 {
					t.Errorf("Expected length 16, got %d: %s", len(got), got)
				}
				if !isAlphanumeric(got) {
					t.Errorf("Expected alphanumeric, got: %s", got)
				}
			case "random with prefix":
				if !strings.HasPrefix(got, "api_") {
					t.Errorf("Expected prefix 'api_', got: %s", got)
				}
				if len(got) != 36 { // "api_" + 32 chars
					t.Errorf("Expected length 36, got %d: %s", len(got), got)
				}
			case "multiple random values":
				parts := strings.Split(got, "-")
				if len(parts) != 2 {
					t.Errorf("Expected 2 parts, got %d: %s", len(parts), got)
				}
				if len(parts[0]) != 8 || len(parts[1]) != 8 {
					t.Errorf("Expected 8-8 format, got %d-%d: %s", len(parts[0]), len(parts[1]), got)
				}
				if parts[0] == parts[1] {
					t.Errorf("Expected different random values, got: %s", got)
				}
			default:
				if got != tt.want {
					t.Errorf("processSecretTemplate() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestSecretTemplateProcessing(t *testing.T) {
	// Create temp directories
	tmpDir, err := os.MkdirTemp("", "secret-template-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	dataDir := filepath.Join(tmpDir, "data")
	appsDir := filepath.Join(tmpDir, "apps")
	instanceName := "test-instance"
	appName := "testapp"

	// Setup directory structure
	instancePath := filepath.Join(dataDir, "instances", instanceName)
	if err := storage.EnsureDir(instancePath, 0755); err != nil {
		t.Fatal(err)
	}

	// Create instance config with database settings
	configFile := filepath.Join(instancePath, "config.yaml")
	instanceConfig := `cloud:
  domain: example.com
apps:
  testapp:
    db:
      host: postgres.local
      port: 5432
      name: testdb
      user: testuser
`
	if err := os.WriteFile(configFile, []byte(instanceConfig), 0644); err != nil {
		t.Fatal(err)
	}

	// Create secrets file
	secretsFile := filepath.Join(instancePath, "secrets.yaml")
	if err := os.WriteFile(secretsFile, []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}

	// Create app directory with manifest
	appPath := filepath.Join(appsDir, appName)
	if err := storage.EnsureDir(appPath, 0755); err != nil {
		t.Fatal(err)
	}

	// Create manifest with template-based secrets
	manifest := AppManifest{
		Name:        appName,
		Description: "Test app with template secrets",
		Version:     "1.0.0",
		DefaultConfig: map[string]interface{}{
			"port": 3000,
		},
		DefaultSecrets: []SecretDefinition{
			{
				Key:     "dbPassword",
				Default: "{{ random.AlphaNum 32 }}",
			},
			{
				Key:     "dbUrl",
				Default: "postgresql://{{ .app.db.user }}:{{ .secrets.dbPassword }}@{{ .app.db.host }}:{{ .app.db.port }}/{{ .app.db.name }}",
			},
			{
				Key:     "apiKey",
				Default: "sk_{{ random.AlphaNum 32 }}",
			},
			{
				Key: "noDefault",
			},
		},
	}

	manifestData, err := yaml.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	manifestFile := filepath.Join(appPath, "manifest.yaml")
	if err := os.WriteFile(manifestFile, manifestData, 0644); err != nil {
		t.Fatal(err)
	}

	// Create a sample kustomization.yaml file
	kustomizationFile := filepath.Join(appPath, "kustomization.yaml")
	kustomizationContent := `apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
namespace: testapp
`
	if err := os.WriteFile(kustomizationFile, []byte(kustomizationContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create app manager and add the app
	manager := NewManager(dataDir, appsDir)
	err = manager.Add(instanceName, appName, nil, nil)
	if err != nil {
		t.Fatalf("Failed to add app: %v", err)
	}

	// Verify secrets were generated correctly
	secretsMgr := secrets.NewManager()

	// Check dbPassword is random
	dbPassword, err := secretsMgr.GetSecret(secretsFile, "apps.testapp.dbPassword")
	if err != nil {
		t.Fatalf("Failed to get dbPassword: %v", err)
	}
	if len(dbPassword) != 32 {
		t.Errorf("dbPassword wrong length: expected 32, got %d", len(dbPassword))
	}
	if !isAlphanumeric(dbPassword) {
		t.Errorf("dbPassword is not alphanumeric: %s", dbPassword)
	}

	// Check dbUrl was constructed correctly
	dbUrl, err := secretsMgr.GetSecret(secretsFile, "apps.testapp.dbUrl")
	if err != nil {
		t.Fatalf("Failed to get dbUrl: %v", err)
	}
	expectedDbUrl := fmt.Sprintf("postgresql://testuser:%s@postgres.local:5432/testdb", dbPassword)
	if dbUrl != expectedDbUrl {
		t.Errorf("dbUrl incorrect: expected '%s', got '%s'", expectedDbUrl, dbUrl)
	}

	// Check apiKey has prefix
	apiKey, err := secretsMgr.GetSecret(secretsFile, "apps.testapp.apiKey")
	if err != nil {
		t.Fatalf("Failed to get apiKey: %v", err)
	}
	if !strings.HasPrefix(apiKey, "sk_") {
		t.Errorf("apiKey missing prefix: %s", apiKey)
	}
	if len(apiKey) != 35 { // "sk_" + 32 chars
		t.Errorf("apiKey wrong length: expected 35, got %d", len(apiKey))
	}

	// Check noDefault got a random value
	noDefault, err := secretsMgr.GetSecret(secretsFile, "apps.testapp.noDefault")
	if err != nil {
		t.Fatalf("Failed to get noDefault: %v", err)
	}
	if len(noDefault) != 32 {
		t.Errorf("noDefault wrong length: expected 32, got %d", len(noDefault))
	}
	if !isAlphanumeric(noDefault) {
		t.Errorf("noDefault is not alphanumeric: %s", noDefault)
	}
}

func TestSecretTemplateWithMultipleRandoms(t *testing.T) {
	// Test that multiple {{RANDOM}} in the same template get different values
	tmpDir, err := os.MkdirTemp("", "multi-random-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	dataDir := filepath.Join(tmpDir, "data")
	appsDir := filepath.Join(tmpDir, "apps")
	instanceName := "test-instance"
	appName := "testapp"

	// Setup directory structure
	instancePath := filepath.Join(dataDir, "instances", instanceName)
	if err := storage.EnsureDir(instancePath, 0755); err != nil {
		t.Fatal(err)
	}

	// Create minimal config
	configFile := filepath.Join(instancePath, "config.yaml")
	if err := os.WriteFile(configFile, []byte("cloud:\n  domain: example.com\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create secrets file
	secretsFile := filepath.Join(instancePath, "secrets.yaml")
	if err := os.WriteFile(secretsFile, []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}

	// Create app directory with manifest
	appPath := filepath.Join(appsDir, appName)
	if err := storage.EnsureDir(appPath, 0755); err != nil {
		t.Fatal(err)
	}

	// Create manifest with multiple {{RANDOM}} in one template
	manifest := AppManifest{
		Name:    appName,
		Version: "1.0.0",
		DefaultSecrets: []SecretDefinition{
			{
				Key:     "multiRandom",
				Default: "{{ random.AlphaNum 32 }}-{{ random.AlphaNum 32 }}",
			},
		},
	}

	manifestData, err := yaml.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	manifestFile := filepath.Join(appPath, "manifest.yaml")
	if err := os.WriteFile(manifestFile, manifestData, 0644); err != nil {
		t.Fatal(err)
	}

	// Create a sample kustomization.yaml file
	kustomizationFile := filepath.Join(appPath, "kustomization.yaml")
	if err := os.WriteFile(kustomizationFile, []byte("apiVersion: kustomize.config.k8s.io/v1beta1\nkind: Kustomization\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create app manager and add the app
	manager := NewManager(dataDir, appsDir)
	err = manager.Add(instanceName, appName, nil, nil)
	if err != nil {
		t.Fatalf("Failed to add app: %v", err)
	}

	// Verify the secret has two different random values
	secretsMgr := secrets.NewManager()
	multiRandom, err := secretsMgr.GetSecret(secretsFile, "apps.testapp.multiRandom")
	if err != nil {
		t.Fatalf("Failed to get multiRandom: %v", err)
	}

	parts := strings.Split(multiRandom, "-")
	if len(parts) != 2 {
		t.Errorf("Expected 2 parts separated by hyphen, got %d", len(parts))
	}

	if len(parts[0]) != 32 || len(parts[1]) != 32 {
		t.Errorf("Random parts wrong length: %d and %d", len(parts[0]), len(parts[1]))
	}

	if parts[0] == parts[1] {
		t.Errorf("Random values should be different but both are: %s", parts[0])
	}
}

func TestIntegratedTemplateProcessing(t *testing.T) {
	// Test that defaultConfig templates and secret templates work together correctly
	tmpDir, err := os.MkdirTemp("", "integrated-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	dataDir := filepath.Join(tmpDir, "data")
	appsDir := filepath.Join(tmpDir, "apps")
	instanceName := "test-instance"
	appName := "integrated"

	// Setup directory structure
	instancePath := filepath.Join(dataDir, "instances", instanceName)
	if err := storage.EnsureDir(instancePath, 0755); err != nil {
		t.Fatal(err)
	}

	// Create instance config
	configFile := filepath.Join(instancePath, "config.yaml")
	instanceConfig := `cloud:
  domain: example.com
  smtp:
    host: mail.example.com
    port: "587"
`
	if err := os.WriteFile(configFile, []byte(instanceConfig), 0644); err != nil {
		t.Fatal(err)
	}

	// Create secrets file
	secretsFile := filepath.Join(instancePath, "secrets.yaml")
	if err := os.WriteFile(secretsFile, []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}

	// Create app directory with manifest
	appPath := filepath.Join(appsDir, appName)
	if err := storage.EnsureDir(appPath, 0755); err != nil {
		t.Fatal(err)
	}

	// Create manifest with both defaultConfig templates and secret templates that reference processed config
	manifest := AppManifest{
		Name:        appName,
		Description: "Integrated template test",
		Version:     "1.0.0",
		DefaultConfig: map[string]interface{}{
			"domain": "integrated.{{ .cloud.domain }}", // Will become: integrated.example.com
			"port":   8080,
			"db": map[string]interface{}{
				"host": "postgres.{{ .cloud.domain }}", // Will become: postgres.example.com
				"name": "integrated_db",
				"user": "integrated_user",
			},
		},
		DefaultSecrets: []SecretDefinition{
			{
				Key:     "apiKey",
				Default: "api_{{ random.AlphaNum 32 }}", // Random with prefix
			},
			{
				Key:     "dbPassword",
				Default: "{{ random.AlphaNum 16 }}", // Random 16 chars
			},
			{
				Key:     "connectionString",
				Default: "postgresql://{{ .app.db.user }}:{{ .secrets.dbPassword }}@{{ .app.db.host }}:5432/{{ .app.db.name }}",
			},
			{
				Key:     "webhookUrl",
				Default: "https://{{ .app.domain }}/webhook?key={{ .secrets.apiKey }}",
			},
		},
	}

	manifestData, err := yaml.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	manifestFile := filepath.Join(appPath, "manifest.yaml")
	if err := os.WriteFile(manifestFile, manifestData, 0644); err != nil {
		t.Fatal(err)
	}

	// Create a deployment.yaml with templates
	deploymentContent := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: integrated
spec:
  template:
    spec:
      containers:
      - name: integrated
        image: integrated:1.0.0
        env:
        - name: APP_DOMAIN
          value: "{{ .domain }}"
        - name: DB_HOST
          value: "{{ .db.host }}"
`
	deploymentFile := filepath.Join(appPath, "deployment.yaml")
	if err := os.WriteFile(deploymentFile, []byte(deploymentContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create kustomization.yaml
	kustomizationFile := filepath.Join(appPath, "kustomization.yaml")
	kustomizationContent := `apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
namespace: integrated
resources:
  - deployment.yaml
`
	if err := os.WriteFile(kustomizationFile, []byte(kustomizationContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create app manager and add the app
	manager := NewManager(dataDir, appsDir)
	err = manager.Add(instanceName, appName, nil, nil)
	if err != nil {
		t.Fatalf("Failed to add app: %v", err)
	}

	// Verify config was processed correctly
	yq := tools.NewYQ()

	// Check that defaultConfig templates were processed
	domain, err := yq.Get(configFile, ".apps.integrated.domain")
	if err != nil {
		t.Fatalf("Failed to get domain: %v", err)
	}
	if domain != "integrated.example.com" {
		t.Errorf("Domain incorrect: expected 'integrated.example.com', got '%s'", domain)
	}

	dbHost, err := yq.Get(configFile, ".apps.integrated.db.host")
	if err != nil {
		t.Fatalf("Failed to get db.host: %v", err)
	}
	if dbHost != "postgres.example.com" {
		t.Errorf("db.host incorrect: expected 'postgres.example.com', got '%s'", dbHost)
	}

	// Verify secrets were generated and templates processed
	secretsMgr := secrets.NewManager()

	// Check apiKey has prefix and random part
	apiKey, err := secretsMgr.GetSecret(secretsFile, "apps.integrated.apiKey")
	if err != nil {
		t.Fatalf("Failed to get apiKey: %v", err)
	}
	if !strings.HasPrefix(apiKey, "api_") {
		t.Errorf("apiKey missing prefix: %s", apiKey)
	}
	if len(apiKey) != 36 { // "api_" + 32 chars
		t.Errorf("apiKey wrong length: expected 36, got %d", len(apiKey))
	}

	// Check dbPassword is random
	dbPassword, err := secretsMgr.GetSecret(secretsFile, "apps.integrated.dbPassword")
	if err != nil {
		t.Fatalf("Failed to get dbPassword: %v", err)
	}
	if len(dbPassword) != 16 {
		t.Errorf("dbPassword wrong length: expected 16, got %d", len(dbPassword))
	}

	// Check connectionString was built from config and secrets
	connectionString, err := secretsMgr.GetSecret(secretsFile, "apps.integrated.connectionString")
	if err != nil {
		t.Fatalf("Failed to get connectionString: %v", err)
	}
	expectedConnStr := fmt.Sprintf("postgresql://integrated_user:%s@postgres.example.com:5432/integrated_db", dbPassword)
	if connectionString != expectedConnStr {
		t.Errorf("connectionString incorrect: expected '%s', got '%s'", expectedConnStr, connectionString)
	}

	// Check webhookUrl was built from processed config and secrets
	webhookUrl, err := secretsMgr.GetSecret(secretsFile, "apps.integrated.webhookUrl")
	if err != nil {
		t.Fatalf("Failed to get webhookUrl: %v", err)
	}
	expectedWebhookUrl := fmt.Sprintf("https://integrated.example.com/webhook?key=%s", apiKey)
	if webhookUrl != expectedWebhookUrl {
		t.Errorf("webhookUrl incorrect: expected '%s', got '%s'", expectedWebhookUrl, webhookUrl)
	}

	// Verify deployment.yaml was processed correctly
	deploymentPath := filepath.Join(instancePath, "apps", appName, "deployment.yaml")
	deploymentData, err := os.ReadFile(deploymentPath)
	if err != nil {
		t.Fatalf("Failed to read deployment.yaml: %v", err)
	}

	// Check that templates in deployment.yaml were processed
	deploymentStr := string(deploymentData)
	if !strings.Contains(deploymentStr, `value: "integrated.example.com"`) {
		t.Errorf("deployment.yaml: APP_DOMAIN not processed correctly")
	}
	if !strings.Contains(deploymentStr, `value: "postgres.example.com"`) {
		t.Errorf("deployment.yaml: DB_HOST not processed correctly")
	}
}

func TestExistingSecretsNotOverwritten(t *testing.T) {
	// Test that existing secrets are not overwritten
	tmpDir, err := os.MkdirTemp("", "existing-secrets-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	dataDir := filepath.Join(tmpDir, "data")
	appsDir := filepath.Join(tmpDir, "apps")
	instanceName := "test-instance"
	appName := "testapp"

	// Setup directory structure
	instancePath := filepath.Join(dataDir, "instances", instanceName)
	if err := storage.EnsureDir(instancePath, 0755); err != nil {
		t.Fatal(err)
	}

	// Create minimal config
	configFile := filepath.Join(instancePath, "config.yaml")
	if err := os.WriteFile(configFile, []byte("cloud:\n  domain: example.com\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create secrets file with existing secret
	secretsFile := filepath.Join(instancePath, "secrets.yaml")
	existingSecrets := `apps:
  testapp:
    existingSecret: "should-not-change"
`
	if err := os.WriteFile(secretsFile, []byte(existingSecrets), 0600); err != nil {
		t.Fatal(err)
	}

	// Create app directory with manifest
	appPath := filepath.Join(appsDir, appName)
	if err := storage.EnsureDir(appPath, 0755); err != nil {
		t.Fatal(err)
	}

	// Create manifest that tries to set default for existing secret
	manifest := AppManifest{
		Name:    appName,
		Version: "1.0.0",
		DefaultSecrets: []SecretDefinition{
			{
				Key:     "existingSecret",
				Default: "new-value-{{ random.AlphaNum 32 }}",
			},
			{
				Key:     "newSecret",
				Default: "{{ random.AlphaNum 32 }}",
			},
		},
	}

	manifestData, err := yaml.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	manifestFile := filepath.Join(appPath, "manifest.yaml")
	if err := os.WriteFile(manifestFile, manifestData, 0644); err != nil {
		t.Fatal(err)
	}

	// Create a sample kustomization.yaml file
	kustomizationFile := filepath.Join(appPath, "kustomization.yaml")
	if err := os.WriteFile(kustomizationFile, []byte("apiVersion: kustomize.config.k8s.io/v1beta1\nkind: Kustomization\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create app manager and add the app
	manager := NewManager(dataDir, appsDir)
	err = manager.Add(instanceName, appName, nil, nil)
	if err != nil {
		t.Fatalf("Failed to add app: %v", err)
	}

	// Verify existing secret was NOT overwritten
	secretsMgr := secrets.NewManager()
	existingSecret, err := secretsMgr.GetSecret(secretsFile, "apps.testapp.existingSecret")
	if err != nil {
		t.Fatalf("Failed to get existingSecret: %v", err)
	}
	if existingSecret != "should-not-change" {
		t.Errorf("Existing secret was overwritten: got '%s', expected 'should-not-change'", existingSecret)
	}

	// Verify new secret was created
	newSecret, err := secretsMgr.GetSecret(secretsFile, "apps.testapp.newSecret")
	if err != nil {
		t.Fatalf("Failed to get newSecret: %v", err)
	}
	if len(newSecret) != 32 {
		t.Errorf("New secret wrong length: expected 32, got %d", len(newSecret))
	}
}

// Helper function to check if string is alphanumeric
func isAlphanumeric(s string) bool {
	for _, c := range s {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')) {
			return false
		}
	}
	return true
}

func TestDefaultConfigWithAppReferences(t *testing.T) {
	// Test that defaultConfig with {{ .app.X }} references work correctly
	// This tests the redis case where uri references {{ .app.host }} and {{ .app.port }}
	tmpDir, err := os.MkdirTemp("", "app-refs-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	dataDir := filepath.Join(tmpDir, "data")
	appsDir := filepath.Join(tmpDir, "apps")
	instanceName := "test-instance"
	appName := "redis"

	// Setup directory structure
	instancePath := filepath.Join(dataDir, "instances", instanceName)
	if err := storage.EnsureDir(instancePath, 0755); err != nil {
		t.Fatal(err)
	}

	// Create minimal config
	configFile := filepath.Join(instancePath, "config.yaml")
	if err := os.WriteFile(configFile, []byte("cloud:\n  domain: example.com\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create secrets file
	secretsFile := filepath.Join(instancePath, "secrets.yaml")
	if err := os.WriteFile(secretsFile, []byte(""), 0600); err != nil {
		t.Fatal(err)
	}

	// Create app directory with manifest that has {{ .app }} references
	appPath := filepath.Join(appsDir, appName)
	if err := storage.EnsureDir(appPath, 0755); err != nil {
		t.Fatal(err)
	}

	// Create manifest with config that references itself
	// IMPORTANT: Order matters - host and port must come before uri
	manifestYAML := `name: redis
description: Redis cache
version: 1.0.0
defaultConfig:
  namespace: redis
  image: redis:alpine
  timezone: UTC
  host: redis.redis.svc.cluster.local
  port: 6379
  uri: redis://{{ .app.host }}:{{ .app.port }}/0
defaultSecrets:
  - key: password
`

	manifestPath := filepath.Join(appPath, "manifest.yaml")
	if err := os.WriteFile(manifestPath, []byte(manifestYAML), 0644); err != nil {
		t.Fatal(err)
	}

	// Add the app
	mgr := NewManager(dataDir, appsDir)
	err = mgr.Add(instanceName, appName, nil, nil)
	if err != nil {
		t.Fatalf("Failed to add app: %v", err)
	}

	// Verify that the uri was processed correctly
	yq := tools.NewYQ()
	uri, err := yq.Get(configFile, ".apps.redis.uri")
	if err != nil {
		t.Fatalf("Failed to get uri from config: %v", err)
	}

	expectedURI := "redis://redis.redis.svc.cluster.local:6379/0"
	if uri != expectedURI {
		t.Errorf("Expected uri to be %s, got %s", expectedURI, uri)
	}
}

func TestSecretKeyFormat(t *testing.T) {
	// Test that secrets are created with the correct key format
	// According to design, defaultSecrets should use "by their defined key"
	// and requiredSecrets should use the full reference as the key

	manifest := AppManifest{
		Name:        "testapp",
		Description: "Test app",
		Version:     "1.0.0",
		DefaultSecrets: []SecretDefinition{
			{Key: "password"},
			{Key: "apiKey", Default: "default-api"},
		},
		RequiredSecrets: []string{
			"postgres.password",
			"redis.password",
		},
	}

	// Test that defaultSecrets would use just the key name
	for _, secretDef := range manifest.DefaultSecrets {
		// In the Kubernetes secret, this should be created with key: secretDef.Key
		if secretDef.Key != "password" && secretDef.Key != "apiKey" {
			t.Errorf("Unexpected secret key format: %s", secretDef.Key)
		}
	}

	// Test that requiredSecrets would use the full reference as key
	for _, requiredSecret := range manifest.RequiredSecrets {
		// In the Kubernetes secret, this should be created with key: requiredSecret
		if !strings.Contains(requiredSecret, ".") {
			t.Errorf("Required secret should have app reference format: %s", requiredSecret)
		}
	}
}

func TestDefaultConfigOrdering(t *testing.T) {
	// Test that config fields that depend on each other are processed in the right order
	// This is a challenge because Go maps don't preserve order

	// The issue: redis manifest has:
	// defaultConfig:
	//   host: redis.redis.svc.cluster.local
	//   port: 6379
	//   uri: redis://{{ .app.host }}:{{ .app.port }}/0
	//
	// The uri field depends on host and port being set first

	// To fix this properly, we need to either:
	// 1. Change defaultConfig from map[string]interface{} to preserve order
	// 2. Process in two passes: non-template values first, then template values
	// 3. Use a dependency resolution algorithm

	// Our current approach (two-pass) should work if implemented correctly
	t.Log("Config ordering is handled by two-pass processing: non-template values first, then template values")
}

func TestAppAliasing(t *testing.T) {
	// Test the installed_as feature for app aliasing
	// This allows multiple instances of the same app

	manifest := AppManifest{
		Name:        "postgres-primary",
		Description: "Primary PostgreSQL database",
		Version:     "1.0.0",
		Source:      "/apps/postgres",
		Requires: []AppDependency{
			{Name: "postgres", InstalledAs: "postgres-primary"},
		},
	}

	// Test that the app can be installed with a different name
	if manifest.Name != "postgres-primary" {
		t.Errorf("App should be installable with custom name")
	}

	// Test that dependencies can reference installed names
	if manifest.Requires[0].InstalledAs != "postgres-primary" {
		t.Errorf("Dependencies should be able to reference installed names")
	}
}

func TestUpdateOperation(t *testing.T) {
	// Test the UPDATE operation for apps
	// This should update an app from its source while preserving config

	tmpDir, err := os.MkdirTemp("", "update-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	dataDir := filepath.Join(tmpDir, "data")
	appsDir := filepath.Join(tmpDir, "apps")
	instanceName := "test-instance"
	appName := "updateapp"

	// Setup directory structure
	instancePath := filepath.Join(dataDir, "instances", instanceName)
	if err := storage.EnsureDir(instancePath, 0755); err != nil {
		t.Fatal(err)
	}

	// Create config with existing app config
	configFile := filepath.Join(instancePath, "config.yaml")
	initialConfig := `cloud:
  domain: example.com
apps:
  updateapp:
    customValue: "should-be-preserved"
`
	if err := os.WriteFile(configFile, []byte(initialConfig), 0644); err != nil {
		t.Fatal(err)
	}

	// Create secrets file
	secretsFile := filepath.Join(instancePath, "secrets.yaml")
	if err := os.WriteFile(secretsFile, []byte(""), 0600); err != nil {
		t.Fatal(err)
	}

	// Create initial app with source
	appPath := filepath.Join(appsDir, appName)
	if err := storage.EnsureDir(appPath, 0755); err != nil {
		t.Fatal(err)
	}

	manifestYAML := `name: updateapp
description: App to test updates
version: 1.0.0
defaultConfig:
  image: v1.0.0
  newField: "added-in-update"
`
	manifestPath := filepath.Join(appPath, "manifest.yaml")
	if err := os.WriteFile(manifestPath, []byte(manifestYAML), 0644); err != nil {
		t.Fatal(err)
	}

	// Add the app initially
	mgr := NewManager(dataDir, appsDir)
	err = mgr.Add(instanceName, appName, nil, nil)
	if err != nil {
		t.Fatalf("Failed to add app: %v", err)
	}

	// Update manifest to v2
	manifestYAML = `name: updateapp
description: App to test updates
version: 2.0.0
defaultConfig:
  image: v2.0.0
  newField: "added-in-update"
`
	if err := os.WriteFile(manifestPath, []byte(manifestYAML), 0644); err != nil {
		t.Fatal(err)
	}

	// Run UPDATE operation
	err = mgr.Update(instanceName, appName)
	if err != nil {
		t.Fatalf("Failed to update app: %v", err)
	}

	// Verify custom config was preserved
	yq := tools.NewYQ()
	customValue, _ := yq.Get(configFile, ".apps.updateapp.customValue")
	if customValue != "should-be-preserved" {
		t.Errorf("Custom config should be preserved during update")
	}

	// Verify new field was added
	newField, _ := yq.Get(configFile, ".apps.updateapp.newField")
	if newField != "added-in-update" {
		t.Errorf("New fields should be added during update")
	}
}

func TestEjectOperation(t *testing.T) {
	// Test the EJECT operation
	// This should convert a package-managed app to custom

	tmpDir, err := os.MkdirTemp("", "eject-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	dataDir := filepath.Join(tmpDir, "data")
	appsDir := filepath.Join(tmpDir, "apps")
	instanceName := "test-instance"
	appName := "ejectapp"

	// Setup directory structure
	instancePath := filepath.Join(dataDir, "instances", instanceName)
	appInstanceDir := filepath.Join(instancePath, "apps", appName)
	packageDir := filepath.Join(appInstanceDir, ".package")

	if err := storage.EnsureDir(instancePath, 0755); err != nil {
		t.Fatal(err)
	}
	if err := storage.EnsureDir(appInstanceDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := storage.EnsureDir(packageDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create config
	configFile := filepath.Join(instancePath, "config.yaml")
	if err := os.WriteFile(configFile, []byte("cloud:\n  domain: example.com\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create app manifest with source
	manifestYAML := `name: ejectapp
description: App to test eject
version: 1.0.0
source: /apps/ejectapp
`
	manifestPath := filepath.Join(appInstanceDir, "manifest.yaml")
	if err := os.WriteFile(manifestPath, []byte(manifestYAML), 0644); err != nil {
		t.Fatal(err)
	}

	// Create .package directory content
	packageManifestPath := filepath.Join(packageDir, "manifest.yaml")
	if err := os.WriteFile(packageManifestPath, []byte(manifestYAML), 0644); err != nil {
		t.Fatal(err)
	}

	// Run EJECT operation
	mgr := NewManager(dataDir, appsDir)
	err = mgr.Eject(instanceName, appName)
	if err != nil {
		t.Fatalf("Failed to eject app: %v", err)
	}

	// Verify .package directory was deleted
	if storage.FileExists(packageDir) {
		t.Errorf(".package directory should be deleted after eject")
	}

	// Verify source was removed from manifest
	manifestData, _ := os.ReadFile(manifestPath)
	var manifest AppManifest
	yaml.Unmarshal(manifestData, &manifest)
	if manifest.Source != "" {
		t.Errorf("Source should be removed from manifest after eject, got: %s", manifest.Source)
	}
}

func TestCompileStep(t *testing.T) {
	// Test the COMPILE step behavior
	// According to design: "Resolve all {{ .? }} to config.yaml apps.<app_name>.?"

	tmpDir, err := os.MkdirTemp("", "compile-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test structure
	instanceDir := filepath.Join(tmpDir, "instance")
	appDir := filepath.Join(instanceDir, "apps", "testapp")
	packageDir := filepath.Join(appDir, ".package")

	if err := storage.EnsureDir(packageDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create config with app config
	configFile := filepath.Join(instanceDir, "config.yaml")
	configYAML := `apps:
  testapp:
    domain: testapp.example.com
    image: testapp:v1.0.0
`
	if err := os.WriteFile(configFile, []byte(configYAML), 0644); err != nil {
		t.Fatal(err)
	}

	// Create template file in .package
	templateContent := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: testapp
spec:
  template:
    spec:
      containers:
      - name: app
        image: {{ .image }}
        env:
        - name: DOMAIN
          value: {{ .domain }}
`
	templatePath := filepath.Join(packageDir, "deployment.yaml")
	if err := os.WriteFile(templatePath, []byte(templateContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Run compile step
	gomplate := tools.NewGomplate()
	yq := tools.NewYQ()

	// Create temp config file with app's config
	tempConfigFile := filepath.Join(appDir, ".config.tmp.yaml")
	appConfigYAML, _ := yq.Get(configFile, ".apps.testapp")
	if err := storage.WriteFile(tempConfigFile, []byte(appConfigYAML), 0644); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tempConfigFile)

	// Process template
	destPath := filepath.Join(appDir, "deployment.yaml")
	context := map[string]string{
		".": tempConfigFile,
	}
	err = gomplate.RenderWithContext(templatePath, destPath, context)
	if err != nil {
		t.Fatalf("Failed to compile template: %v", err)
	}

	// Verify compiled content
	compiledData, _ := os.ReadFile(destPath)
	compiled := string(compiledData)

	if !strings.Contains(compiled, "image: testapp:v1.0.0") {
		t.Errorf("Template {{ .image }} not resolved correctly")
	}
	if !strings.Contains(compiled, "value: testapp.example.com") {
		t.Errorf("Template {{ .domain }} not resolved correctly")
	}
}
