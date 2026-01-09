package config

import (
	"reflect"
	"testing"
)

func TestGetValue(t *testing.T) {
	config := map[string]interface{}{
		"cloud": map[string]interface{}{
			"domain": "example.com",
			"name":   "test-cloud",
		},
		"simple": "value",
	}

	tests := []struct {
		name     string
		path     string
		expected interface{}
	}{
		{"simple path", "simple", "value"},
		{"nested path", "cloud.domain", "example.com"},
		{"nested path 2", "cloud.name", "test-cloud"},
		{"missing path", "missing", nil},
		{"missing nested", "cloud.missing", nil},
		{"invalid nested", "simple.nested", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetValue(config, tt.path)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("GetValue(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestValidatePaths(t *testing.T) {
	config := map[string]interface{}{
		"cloud": map[string]interface{}{
			"domain": "example.com",
			"name":   "test-cloud",
		},
		"network": map[string]interface{}{
			"subnet": "10.0.0.0/24",
		},
	}

	tests := []struct {
		name     string
		paths    []string
		expected []string
	}{
		{
			name:     "all paths exist",
			paths:    []string{"cloud.domain", "cloud.name", "network.subnet"},
			expected: nil,
		},
		{
			name:     "some paths missing",
			paths:    []string{"cloud.domain", "missing.path", "network.missing"},
			expected: []string{"missing.path", "network.missing"},
		},
		{
			name:     "all paths missing",
			paths:    []string{"foo.bar", "baz.qux"},
			expected: []string{"foo.bar", "baz.qux"},
		},
		{
			name:     "empty paths",
			paths:    []string{},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidatePaths(config, tt.paths)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("ValidatePaths() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestExpandTemplate(t *testing.T) {
	config := map[string]interface{}{
		"cloud": map[string]interface{}{
			"domain": "example.com",
			"name":   "test-cloud",
		},
		"port": 8080,
	}

	tests := []struct {
		name      string
		template  string
		expected  string
		shouldErr bool
	}{
		{
			name:     "simple template",
			template: "registry.{{ .cloud.domain }}",
			expected: "registry.example.com",
		},
		{
			name:     "multiple templates",
			template: "{{ .cloud.name }}.{{ .cloud.domain }}",
			expected: "test-cloud.example.com",
		},
		{
			name:     "no template",
			template: "plain-string",
			expected: "plain-string",
		},
		{
			name:     "template with text",
			template: "http://{{ .cloud.domain }}:{{ .port }}/api",
			expected: "http://example.com:8080/api",
		},
		{
			name:      "invalid template syntax",
			template:  "{{ .cloud.domain",
			shouldErr: true,
		},
		{
			name:     "missing value renders as no value",
			template: "{{ .missing.path }}",
			expected: "<no value>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ExpandTemplate(tt.template, config)
			if tt.shouldErr {
				if err == nil {
					t.Errorf("ExpandTemplate() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("ExpandTemplate() unexpected error: %v", err)
				return
			}
			if result != tt.expected {
				t.Errorf("ExpandTemplate() = %q, want %q", result, tt.expected)
			}
		})
	}
}
