// Package prompt provides simple utilities for interactive CLI prompts
package prompt

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// String prompts the user for a string value with a default
func String(prompt, defaultValue string) (string, error) {
	if defaultValue != "" {
		fmt.Printf("%s [%s]: ", prompt, defaultValue)
	} else {
		fmt.Printf("%s: ", prompt)
	}

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read input: %w", err)
	}

	input = strings.TrimSpace(input)
	if input == "" {
		return defaultValue, nil
	}

	return input, nil
}

// Int prompts the user for an integer value with a default
func Int(prompt string, defaultValue int) (int, error) {
	fmt.Printf("%s [%d]: ", prompt, defaultValue)

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return 0, fmt.Errorf("failed to read input: %w", err)
	}

	input = strings.TrimSpace(input)
	if input == "" {
		return defaultValue, nil
	}

	value, err := strconv.Atoi(input)
	if err != nil {
		return 0, fmt.Errorf("invalid integer value: %s", input)
	}

	return value, nil
}

// Bool prompts the user for a boolean value with a default
func Bool(prompt string, defaultValue bool) (bool, error) {
	defaultStr := "y/n"
	if defaultValue {
		defaultStr = "Y/n"
	} else {
		defaultStr = "y/N"
	}

	fmt.Printf("%s [%s]: ", prompt, defaultStr)

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return false, fmt.Errorf("failed to read input: %w", err)
	}

	input = strings.TrimSpace(input)
	input = strings.ToLower(input)

	if input == "" {
		return defaultValue, nil
	}

	switch input {
	case "y", "yes", "true":
		return true, nil
	case "n", "no", "false":
		return false, nil
	default:
		return false, fmt.Errorf("invalid boolean value: %s (expected y/n/yes/no/true/false)", input)
	}
}
