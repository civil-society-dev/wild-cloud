package config_test

import (
	"fmt"

	"github.com/wild-cloud/wild-central/wild/internal/config"
)

func ExampleGetValue() {
	cfg := map[string]interface{}{
		"cloud": map[string]interface{}{
			"domain": "example.com",
		},
	}

	value := config.GetValue(cfg, "cloud.domain")
	fmt.Println(value)
	// Output: example.com
}

func ExampleValidatePaths() {
	cfg := map[string]interface{}{
		"cloud": map[string]interface{}{
			"domain": "example.com",
		},
	}

	required := []string{"cloud.domain", "cloud.name", "network.subnet"}
	missing := config.ValidatePaths(cfg, required)
	fmt.Printf("Missing paths: %v\n", missing)
	// Output: Missing paths: [cloud.name network.subnet]
}

func ExampleExpandTemplate() {
	cfg := map[string]interface{}{
		"cloud": map[string]interface{}{
			"domain": "example.com",
		},
	}

	result, _ := config.ExpandTemplate("registry.{{ .cloud.domain }}", cfg)
	fmt.Println(result)
	// Output: registry.example.com
}
