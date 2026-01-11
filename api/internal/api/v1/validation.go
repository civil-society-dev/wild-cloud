package v1

import (
	"fmt"
	"regexp"
)

// validNamePattern matches valid resource names.
// Names must:
// - Start and end with a lowercase letter or number
// - Contain only lowercase letters, numbers, and hyphens
// - Be between 1 and 63 characters
// This follows Kubernetes naming conventions for consistency.
var validNamePattern = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`)

// ValidateName validates a resource name (instance, app, node, etc.).
// Returns an error if the name is invalid.
func ValidateName(name string) error {
	if name == "" {
		return fmt.Errorf("name is required")
	}
	if len(name) > 63 {
		return fmt.Errorf("name must be 63 characters or less")
	}
	if !validNamePattern.MatchString(name) {
		return fmt.Errorf("name must contain only lowercase letters, numbers, and hyphens, and must start and end with a letter or number")
	}
	return nil
}

// ValidateInstanceName validates an instance name with a user-friendly error message.
func ValidateInstanceName(name string) error {
	if err := ValidateName(name); err != nil {
		return fmt.Errorf("invalid instance name: %w", err)
	}
	return nil
}

// ValidateAppName validates an app name with a user-friendly error message.
func ValidateAppName(name string) error {
	if err := ValidateName(name); err != nil {
		return fmt.Errorf("invalid app name: %w", err)
	}
	return nil
}

// ValidateServiceName validates a service name with a user-friendly error message.
func ValidateServiceName(name string) error {
	if err := ValidateName(name); err != nil {
		return fmt.Errorf("invalid service name: %w", err)
	}
	return nil
}

// ValidateNodeIdentifier validates a node identifier (hostname or IP).
// Node identifiers have slightly relaxed rules since they can be IPs.
func ValidateNodeIdentifier(identifier string) error {
	if identifier == "" {
		return fmt.Errorf("node identifier is required")
	}
	if len(identifier) > 253 {
		return fmt.Errorf("node identifier must be 253 characters or less")
	}
	// Basic check for path traversal
	if containsPathTraversal(identifier) {
		return fmt.Errorf("invalid node identifier: contains invalid characters")
	}
	return nil
}

// containsPathTraversal checks if a string contains path traversal patterns.
func containsPathTraversal(s string) bool {
	// Check for common path traversal patterns
	dangerous := []string{"..", "/", "\\", "\x00"}
	for _, d := range dangerous {
		for i := 0; i <= len(s)-len(d); i++ {
			if s[i:i+len(d)] == d {
				return true
			}
		}
	}
	return false
}
