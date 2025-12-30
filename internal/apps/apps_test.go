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
	err = manager.Add(instanceName, appName, nil)
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
			template: "postgresql://{{ .config.apps.testapp.db.user }}@{{ .config.apps.testapp.db.host }}",
			want:     "postgresql://testuser@postgres.local",
			wantErr:  false,
		},
		{
			name:     "reference existing secret",
			template: "webhook_url?key={{ .secrets.apps.testapp.existingSecret }}",
			want:     "webhook_url?key=myexistingsecret",
			wantErr:  false,
		},
		{
			name:     "complex template with config and secrets",
			template: "postgresql://{{ .config.apps.testapp.db.user }}:{{ .secrets.apps.testapp.existingSecret }}@{{ .config.apps.testapp.db.host }}:{{ .config.apps.testapp.db.port }}/{{ .config.apps.testapp.db.name }}",
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
			got, err := processSecretTemplate(tt.template, configFile, secretsFile, gomplate)
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
				Key:     "apps.testapp.dbPassword",
				Default: "{{ random.AlphaNum 32 }}",
			},
			{
				Key:     "apps.testapp.dbUrl",
				Default: "postgresql://{{ .config.apps.testapp.db.user }}:{{ .secrets.apps.testapp.dbPassword }}@{{ .config.apps.testapp.db.host }}:{{ .config.apps.testapp.db.port }}/{{ .config.apps.testapp.db.name }}",
			},
			{
				Key:     "apps.testapp.apiKey",
				Default: "sk_{{ random.AlphaNum 32 }}",
			},
			{
				Key: "apps.testapp.noDefault",
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
	err = manager.Add(instanceName, appName, nil)
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
				Key:     "apps.testapp.multiRandom",
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
	err = manager.Add(instanceName, appName, nil)
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
				Key:     "apps.integrated.apiKey",
				Default: "api_{{ random.AlphaNum 32 }}", // Random with prefix
			},
			{
				Key:     "apps.integrated.dbPassword",
				Default: "{{ random.AlphaNum 16 }}", // Random 16 chars
			},
			{
				Key:     "apps.integrated.connectionString",
				Default: "postgresql://{{ .config.apps.integrated.db.user }}:{{ .secrets.apps.integrated.dbPassword }}@{{ .config.apps.integrated.db.host }}:5432/{{ .config.apps.integrated.db.name }}",
			},
			{
				Key:     "apps.integrated.webhookUrl",
				Default: "https://{{ .config.apps.integrated.domain }}/webhook?key={{ .secrets.apps.integrated.apiKey }}",
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
          value: "{{ .apps.integrated.domain }}"
        - name: DB_HOST
          value: "{{ .apps.integrated.db.host }}"
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
	err = manager.Add(instanceName, appName, nil)
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
				Key:     "apps.testapp.existingSecret",
				Default: "new-value-{{ random.AlphaNum 32 }}",
			},
			{
				Key:     "apps.testapp.newSecret",
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
	err = manager.Add(instanceName, appName, nil)
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
