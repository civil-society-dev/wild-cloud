package tools

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// YQ provides a wrapper around the yq command-line tool
type YQ struct {
	yqPath string
}

// NewYQ creates a new YQ wrapper
func NewYQ() *YQ {
	// Find yq in PATH
	path, err := exec.LookPath("yq")
	if err != nil {
		// Default to "yq" and let exec handle the error
		path = "yq"
	}
	return &YQ{yqPath: path}
}

// Get retrieves a value from a YAML file using a yq expression
func (y *YQ) Get(filePath, expression string) (string, error) {
	cmd := exec.Command(y.yqPath, expression, filePath)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("yq get failed: %w, stderr: %s", err, stderr.String())
	}

	return strings.TrimSpace(stdout.String()), nil
}

// Set sets a value in a YAML file using a yq expression
func (y *YQ) Set(filePath, expression, value string) error {
	// yq -i '.path = "value"' file.yaml
	// Ensure expression starts with '.' for yq v4 syntax
	if !strings.HasPrefix(expression, ".") {
		expression = "." + expression
	}

	// Properly quote the value to handle special characters
	quotedValue := fmt.Sprintf(`"%s"`, strings.ReplaceAll(value, `"`, `\"`))
	setExpr := fmt.Sprintf("%s = %s", expression, quotedValue)
	cmd := exec.Command(y.yqPath, "-i", setExpr, filePath)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("yq set failed: %w, stderr: %s", err, stderr.String())
	}

	return nil
}

// Merge merges two YAML files
func (y *YQ) Merge(file1, file2, outputFile string) error {
	// yq eval-all '. as $item ireduce ({}; . * $item)' file1.yaml file2.yaml > output.yaml
	cmd := exec.Command(y.yqPath, "eval-all", ". as $item ireduce ({}; . * $item)", file1, file2)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("yq merge failed: %w, stderr: %s", err, stderr.String())
	}

	// Write output
	return exec.Command("sh", "-c", fmt.Sprintf("echo '%s' > %s", stdout.String(), outputFile)).Run()
}

// Delete removes a key from a YAML file
func (y *YQ) Delete(filePath, expression string) error {
	// yq -i 'del(.path)' file.yaml
	delExpr := fmt.Sprintf("del(%s)", expression)
	cmd := exec.Command(y.yqPath, "-i", delExpr, filePath)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("yq delete failed: %w, stderr: %s", err, stderr.String())
	}

	return nil
}

// Validate checks if a YAML file is valid
func (y *YQ) Validate(filePath string) error {
	cmd := exec.Command(y.yqPath, "eval", ".", filePath)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("yaml validation failed: %w, stderr: %s", err, stderr.String())
	}

	return nil
}

// Exec executes an arbitrary yq expression on a file
func (y *YQ) Exec(args ...string) ([]byte, error) {
	cmd := exec.Command(y.yqPath, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("yq exec failed: %w, stderr: %s", err, stderr.String())
	}

	return stdout.Bytes(), nil
}

// CleanYQOutput removes trailing newlines and "null" values from yq output
func CleanYQOutput(output string) string {
	output = strings.TrimSpace(output)
	if output == "null" {
		return ""
	}
	return output
}
