package tools

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// Gomplate provides a wrapper around the gomplate command-line tool
type Gomplate struct {
	gomplatePath string
}

// NewGomplate creates a new Gomplate wrapper
func NewGomplate() *Gomplate {
	// Find gomplate in PATH
	path, err := exec.LookPath("gomplate")
	if err != nil {
		// Default to "gomplate" and let exec handle the error
		path = "gomplate"
	}
	return &Gomplate{gomplatePath: path}
}

// Render renders a template file with the given data sources
func (g *Gomplate) Render(templatePath, outputPath string, dataSources map[string]string) error {
	args := []string{
		"-f", templatePath,
		"-o", outputPath,
	}

	// Add data sources
	for name, path := range dataSources {
		args = append(args, "-d", fmt.Sprintf("%s=%s", name, path))
	}

	cmd := exec.Command(g.gomplatePath, args...)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("gomplate render failed: %w, stderr: %s", err, stderr.String())
	}

	return nil
}

// RenderString renders a template string with the given data sources
func (g *Gomplate) RenderString(template string, dataSources map[string]string) (string, error) {
	args := []string{
		"-i", template,
	}

	// Add data sources
	for name, path := range dataSources {
		args = append(args, "-d", fmt.Sprintf("%s=%s", name, path))
	}

	cmd := exec.Command(g.gomplatePath, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("gomplate render string failed: %w, stderr: %s", err, stderr.String())
	}

	return strings.TrimSpace(stdout.String()), nil
}

// RenderWithContext renders a template with context values passed as arguments
func (g *Gomplate) RenderWithContext(templatePath, outputPath string, context map[string]string) error {
	args := []string{
		"-f", templatePath,
		"-o", outputPath,
	}

	// Add context values
	for key, value := range context {
		args = append(args, "-c", fmt.Sprintf("%s=%s", key, value))
	}

	cmd := exec.Command(g.gomplatePath, args...)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("gomplate render with context failed: %w, stderr: %s", err, stderr.String())
	}

	return nil
}

// Exec executes gomplate with arbitrary arguments
func (g *Gomplate) Exec(args ...string) (string, error) {
	cmd := exec.Command(g.gomplatePath, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("gomplate exec failed: %w, stderr: %s", err, stderr.String())
	}

	return strings.TrimSpace(stdout.String()), nil
}
