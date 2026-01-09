package config

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
)

// ValidatePaths checks if all required paths exist in the config
// Returns a list of missing paths
func ValidatePaths(config map[string]interface{}, paths []string) []string {
	var missing []string
	for _, path := range paths {
		if GetValue(config, path) == nil {
			missing = append(missing, path)
		}
	}
	return missing
}

// GetValue retrieves a nested value from config using dot notation
// Returns nil if the path doesn't exist
func GetValue(config map[string]interface{}, path string) interface{} {
	parts := strings.Split(path, ".")
	var current interface{} = config

	for _, part := range parts {
		m, ok := current.(map[string]interface{})
		if !ok {
			return nil
		}
		current = m[part]
		if current == nil {
			return nil
		}
	}

	return current
}

// ExpandTemplate expands {{ .path.to.value }} templates in the string
// Returns the original string if no templates are present
func ExpandTemplate(tmpl string, config map[string]interface{}) (string, error) {
	// Return original if no template markers
	if !strings.Contains(tmpl, "{{") {
		return tmpl, nil
	}

	t, err := template.New("config").Parse(tmpl)
	if err != nil {
		return "", fmt.Errorf("parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, config); err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}

	return buf.String(), nil
}
