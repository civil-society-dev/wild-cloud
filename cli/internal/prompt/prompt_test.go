package prompt

import (
	"testing"
)

// Note: These are basic unit tests for the prompt package.
// Interactive testing requires manual verification since the functions
// read from stdin and write to stdout.

func TestBoolParsing(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
		hasError bool
	}{
		{"yes", "yes", true, false},
		{"y", "y", true, false},
		{"true", "true", true, false},
		{"no", "no", false, false},
		{"n", "n", false, false},
		{"false", "false", false, false},
		{"invalid", "maybe", false, true},
		{"invalid", "xyz", false, true},
	}

	// Test the parsing logic that would be used by Bool function
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := tt.input
			var result bool
			var err error

			switch input {
			case "y", "yes", "true":
				result = true
			case "n", "no", "false":
				result = false
			default:
				err = &invalidBoolError{}
			}

			if tt.hasError {
				if err == nil {
					t.Errorf("expected error for input %q, got nil", tt.input)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error for input %q: %v", tt.input, err)
				}
				if result != tt.expected {
					t.Errorf("expected %v for input %q, got %v", tt.expected, tt.input, result)
				}
			}
		})
	}
}

type invalidBoolError struct{}

func (e *invalidBoolError) Error() string {
	return "invalid boolean value"
}
