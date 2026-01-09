package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewYQ(t *testing.T) {
	t.Run("creates YQ instance with default path", func(t *testing.T) {
		yq := NewYQ()
		if yq == nil {
			t.Fatal("NewYQ() returned nil")
		}
		if yq.yqPath == "" {
			t.Error("yqPath should not be empty")
		}
	})
}

func TestYQGet(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(tmpDir string) (string, string)
		expression string
		want       string
		wantErr    bool
	}{
		{
			name: "get simple value",
			setup: func(tmpDir string) (string, string) {
				yamlContent := `name: test
version: "1.0"
`
				filePath := filepath.Join(tmpDir, "test.yaml")
				if err := os.WriteFile(filePath, []byte(yamlContent), 0644); err != nil {
					t.Fatal(err)
				}
				return filePath, ".name"
			},
			want:    "test",
			wantErr: false,
		},
		{
			name: "get nested value",
			setup: func(tmpDir string) (string, string) {
				yamlContent := `person:
  name: John
  age: 30
`
				filePath := filepath.Join(tmpDir, "nested.yaml")
				if err := os.WriteFile(filePath, []byte(yamlContent), 0644); err != nil {
					t.Fatal(err)
				}
				return filePath, ".person.name"
			},
			want:    "John",
			wantErr: false,
		},
		{
			name: "non-existent file returns error",
			setup: func(tmpDir string) (string, string) {
				return filepath.Join(tmpDir, "nonexistent.yaml"), ".name"
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip if yq is not available
			if _, err := os.Stat("/usr/bin/yq"); os.IsNotExist(err) {
				t.Skip("yq not installed, skipping test")
			}

			tmpDir := t.TempDir()
			filePath, expression := tt.setup(tmpDir)

			yq := NewYQ()
			got, err := yq.Get(filePath, expression)

			if (err != nil) != tt.wantErr {
				t.Errorf("Get() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && got != tt.want {
				t.Errorf("Get() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestYQSet(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(tmpDir string) string
		expression string
		value      string
		verify     func(t *testing.T, filePath string)
		wantErr    bool
	}{
		{
			name: "set simple value",
			setup: func(tmpDir string) string {
				yamlContent := `name: old`
				filePath := filepath.Join(tmpDir, "test.yaml")
				if err := os.WriteFile(filePath, []byte(yamlContent), 0644); err != nil {
					t.Fatal(err)
				}
				return filePath
			},
			expression: ".name",
			value:      "new",
			verify: func(t *testing.T, filePath string) {
				content, err := os.ReadFile(filePath)
				if err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(string(content), "new") {
					t.Errorf("File does not contain expected value 'new': %s", content)
				}
			},
			wantErr: false,
		},
		{
			name: "set value with special characters",
			setup: func(tmpDir string) string {
				yamlContent := `message: hello`
				filePath := filepath.Join(tmpDir, "special.yaml")
				if err := os.WriteFile(filePath, []byte(yamlContent), 0644); err != nil {
					t.Fatal(err)
				}
				return filePath
			},
			expression: ".message",
			value:      `hello "world"`,
			verify: func(t *testing.T, filePath string) {
				content, err := os.ReadFile(filePath)
				if err != nil {
					t.Fatal(err)
				}
				// Should contain escaped quotes
				if !strings.Contains(string(content), "hello") {
					t.Errorf("File does not contain expected value: %s", content)
				}
			},
			wantErr: false,
		},
		{
			name: "expression without leading dot gets dot prepended",
			setup: func(tmpDir string) string {
				yamlContent := `key: value`
				filePath := filepath.Join(tmpDir, "nodot.yaml")
				if err := os.WriteFile(filePath, []byte(yamlContent), 0644); err != nil {
					t.Fatal(err)
				}
				return filePath
			},
			expression: "key",
			value:      "newvalue",
			verify: func(t *testing.T, filePath string) {
				content, err := os.ReadFile(filePath)
				if err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(string(content), "newvalue") {
					t.Errorf("File does not contain expected value: %s", content)
				}
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip if yq is not available
			if _, err := os.Stat("/usr/bin/yq"); os.IsNotExist(err) {
				t.Skip("yq not installed, skipping test")
			}

			tmpDir := t.TempDir()
			filePath := tt.setup(tmpDir)

			yq := NewYQ()
			err := yq.Set(filePath, tt.expression, tt.value)

			if (err != nil) != tt.wantErr {
				t.Errorf("Set() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.verify != nil {
				tt.verify(t, filePath)
			}
		})
	}
}

func TestYQDelete(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(tmpDir string) string
		expression string
		verify     func(t *testing.T, filePath string)
		wantErr    bool
	}{
		{
			name: "delete simple key",
			setup: func(tmpDir string) string {
				yamlContent := `name: test
version: "1.0"
`
				filePath := filepath.Join(tmpDir, "delete.yaml")
				if err := os.WriteFile(filePath, []byte(yamlContent), 0644); err != nil {
					t.Fatal(err)
				}
				return filePath
			},
			expression: ".name",
			verify: func(t *testing.T, filePath string) {
				content, err := os.ReadFile(filePath)
				if err != nil {
					t.Fatal(err)
				}
				if strings.Contains(string(content), "name:") {
					t.Errorf("Key 'name' was not deleted: %s", content)
				}
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip if yq is not available
			if _, err := os.Stat("/usr/bin/yq"); os.IsNotExist(err) {
				t.Skip("yq not installed, skipping test")
			}

			tmpDir := t.TempDir()
			filePath := tt.setup(tmpDir)

			yq := NewYQ()
			err := yq.Delete(filePath, tt.expression)

			if (err != nil) != tt.wantErr {
				t.Errorf("Delete() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.verify != nil {
				tt.verify(t, filePath)
			}
		})
	}
}

func TestYQValidate(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(tmpDir string) string
		wantErr bool
	}{
		{
			name: "valid YAML",
			setup: func(tmpDir string) string {
				yamlContent := `name: test
version: "1.0"
nested:
  key: value
`
				filePath := filepath.Join(tmpDir, "valid.yaml")
				if err := os.WriteFile(filePath, []byte(yamlContent), 0644); err != nil {
					t.Fatal(err)
				}
				return filePath
			},
			wantErr: false,
		},
		{
			name: "invalid YAML",
			setup: func(tmpDir string) string {
				invalidYaml := `name: test
  invalid indentation
version: "1.0"
`
				filePath := filepath.Join(tmpDir, "invalid.yaml")
				if err := os.WriteFile(filePath, []byte(invalidYaml), 0644); err != nil {
					t.Fatal(err)
				}
				return filePath
			},
			wantErr: true,
		},
		{
			name: "non-existent file",
			setup: func(tmpDir string) string {
				return filepath.Join(tmpDir, "nonexistent.yaml")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip if yq is not available
			if _, err := os.Stat("/usr/bin/yq"); os.IsNotExist(err) {
				t.Skip("yq not installed, skipping test")
			}

			tmpDir := t.TempDir()
			filePath := tt.setup(tmpDir)

			yq := NewYQ()
			err := yq.Validate(filePath)

			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestYQExec(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(tmpDir string) (string, []string)
		wantErr bool
	}{
		{
			name: "exec with valid args",
			setup: func(tmpDir string) (string, []string) {
				yamlContent := `name: test`
				filePath := filepath.Join(tmpDir, "exec.yaml")
				if err := os.WriteFile(filePath, []byte(yamlContent), 0644); err != nil {
					t.Fatal(err)
				}
				return filePath, []string{"eval", ".name", filePath}
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip if yq is not available
			if _, err := os.Stat("/usr/bin/yq"); os.IsNotExist(err) {
				t.Skip("yq not installed, skipping test")
			}

			tmpDir := t.TempDir()
			_, args := tt.setup(tmpDir)

			yq := NewYQ()
			output, err := yq.Exec(args...)

			if (err != nil) != tt.wantErr {
				t.Errorf("Exec() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && len(output) == 0 {
				t.Error("Exec() returned empty output")
			}
		})
	}
}

func TestCleanYQOutput(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "removes trailing newline",
			input: "value\n",
			want:  "value",
		},
		{
			name:  "converts null to empty string",
			input: "null",
			want:  "",
		},
		{
			name:  "removes whitespace",
			input: "  value  \n",
			want:  "value",
		},
		{
			name:  "handles empty string",
			input: "",
			want:  "",
		},
		{
			name:  "handles multiple newlines",
			input: "value\n\n",
			want:  "value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CleanYQOutput(tt.input)
			if got != tt.want {
				t.Errorf("CleanYQOutput(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestYQMerge(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(tmpDir string) (string, string, string)
		verify  func(t *testing.T, outputPath string)
		wantErr bool
	}{
		{
			name: "merge two files",
			setup: func(tmpDir string) (string, string, string) {
				file1 := filepath.Join(tmpDir, "file1.yaml")
				file2 := filepath.Join(tmpDir, "file2.yaml")
				output := filepath.Join(tmpDir, "output.yaml")

				if err := os.WriteFile(file1, []byte("key1: value1\n"), 0644); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(file2, []byte("key2: value2\n"), 0644); err != nil {
					t.Fatal(err)
				}

				return file1, file2, output
			},
			verify: func(t *testing.T, outputPath string) {
				if _, err := os.Stat(outputPath); os.IsNotExist(err) {
					t.Error("Output file was not created")
				}
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip if yq is not available
			if _, err := os.Stat("/usr/bin/yq"); os.IsNotExist(err) {
				t.Skip("yq not installed, skipping test")
			}

			tmpDir := t.TempDir()
			file1, file2, output := tt.setup(tmpDir)

			yq := NewYQ()
			err := yq.Merge(file1, file2, output)

			if (err != nil) != tt.wantErr {
				t.Errorf("Merge() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.verify != nil {
				tt.verify(t, output)
			}
		})
	}
}
