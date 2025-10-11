package setup

import (
	"io/fs"
	"testing"
)

func TestListServices(t *testing.T) {
	services, err := ListServices()
	if err != nil {
		t.Fatalf("ListServices failed: %v", err)
	}

	if len(services) == 0 {
		t.Fatal("Expected at least one service, got none")
	}

	t.Logf("Found %d services: %v", len(services), services)

	// Check for known services
	expectedServices := []string{"traefik", "metallb", "coredns"}
	for _, expected := range expectedServices {
		found := false
		for _, service := range services {
			if service == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected service %q not found in list", expected)
		}
	}
}

func TestServiceExists(t *testing.T) {
	tests := []struct {
		name     string
		service  string
		expected bool
	}{
		{"existing service", "traefik", true},
		{"non-existent service", "nonexistent", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exists := ServiceExists(tt.service)
			if exists != tt.expected {
				t.Errorf("ServiceExists(%q) = %v, want %v", tt.service, exists, tt.expected)
			}
		})
	}
}

func TestGetManifest(t *testing.T) {
	services, err := ListServices()
	if err != nil {
		t.Fatalf("ListServices failed: %v", err)
	}

	if len(services) == 0 {
		t.Skip("No services to test")
	}

	// Test the first service
	serviceName := services[0]
	manifest, err := GetManifest(serviceName)
	if err != nil {
		t.Fatalf("GetManifest(%q) failed: %v", serviceName, err)
	}

	if manifest.Name == "" {
		t.Error("Manifest name is empty")
	}

	t.Logf("Service: %s, Manifest: %+v", serviceName, manifest)
}

func TestGetServiceFile(t *testing.T) {
	// Test getting wild-manifest.yaml
	data, err := GetServiceFile("traefik", "wild-manifest.yaml")
	if err != nil {
		t.Fatalf("GetServiceFile failed: %v", err)
	}

	if len(data) == 0 {
		t.Error("Expected non-empty file data")
	}

	t.Logf("Retrieved wild-manifest.yaml, size: %d bytes", len(data))
}

func TestGetServiceFS(t *testing.T) {
	serviceFS, err := GetServiceFS("traefik")
	if err != nil {
		t.Fatalf("GetServiceFS failed: %v", err)
	}

	// Verify we can read the filesystem
	entries, err := fs.ReadDir(serviceFS, ".")
	if err != nil {
		t.Fatalf("ReadDir failed: %v", err)
	}

	if len(entries) == 0 {
		t.Error("Expected non-empty service directory")
	}

	t.Logf("Service FS entries: %d", len(entries))
	for _, entry := range entries {
		t.Logf("  - %s (dir: %v)", entry.Name(), entry.IsDir())
	}
}

func TestGetKustomizeTemplate(t *testing.T) {
	templateFS, err := GetKustomizeTemplate("traefik")
	if err != nil {
		t.Fatalf("GetKustomizeTemplate failed: %v", err)
	}

	// Verify we can read the template filesystem
	entries, err := fs.ReadDir(templateFS, ".")
	if err != nil {
		t.Fatalf("ReadDir failed: %v", err)
	}

	if len(entries) == 0 {
		t.Error("Expected non-empty kustomize.template directory")
	}

	t.Logf("Kustomize template entries: %d", len(entries))
	for _, entry := range entries {
		t.Logf("  - %s (dir: %v)", entry.Name(), entry.IsDir())
	}
}

func TestGetNonExistentService(t *testing.T) {
	_, err := GetManifest("nonexistent-service")
	if err == nil {
		t.Error("Expected error for non-existent service, got nil")
	}
}

func TestGetClusterNodesFS(t *testing.T) {
	clusterNodesFS, err := GetClusterNodesFS()
	if err != nil {
		t.Fatalf("GetClusterNodesFS failed: %v", err)
	}

	entries, err := fs.ReadDir(clusterNodesFS, ".")
	if err != nil {
		t.Fatalf("ReadDir failed: %v", err)
	}

	if len(entries) == 0 {
		t.Error("Expected non-empty cluster-nodes directory")
	}

	t.Logf("Cluster-nodes entries: %d", len(entries))
	for _, entry := range entries {
		t.Logf("  - %s (dir: %v)", entry.Name(), entry.IsDir())
	}
}
