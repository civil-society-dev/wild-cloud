package prompt_test

import (
	"fmt"
	// "github.com/wild-cloud/wild-central/wild/internal/prompt"
)

// ExampleString demonstrates the String prompt function
func ExampleString() {
	// This example shows the prompt output format
	// Actual usage would read from stdin interactively
	fmt.Println("Enter SMTP host [smtp.gmail.com]:")
	// User input: <empty> (returns default)
	// Result: "smtp.gmail.com"

	fmt.Println("Enter SMTP host [smtp.gmail.com]:")
	// User input: "smtp.example.com"
	// Result: "smtp.example.com"
}

// ExampleInt demonstrates the Int prompt function
func ExampleInt() {
	// This example shows the prompt output format
	fmt.Println("Enter SMTP port [587]:")
	// User input: <empty> (returns default)
	// Result: 587

	fmt.Println("Enter SMTP port [587]:")
	// User input: "465"
	// Result: 465
}

// ExampleBool demonstrates the Bool prompt function
func ExampleBool() {
	// This example shows the prompt output format
	fmt.Println("Enable TLS [Y/n]:")
	// User input: <empty> (returns default true)
	// Result: true

	fmt.Println("Enable TLS [Y/n]:")
	// User input: "n"
	// Result: false

	fmt.Println("Enable debug mode [y/N]:")
	// User input: <empty> (returns default false)
	// Result: false

	fmt.Println("Enable debug mode [y/N]:")
	// User input: "yes"
	// Result: true
}

// Example usage in a real command
func ExampleUsage() {
	// Example of using prompt functions in a CLI command:
	//
	// host, err := prompt.String("Enter SMTP host", "smtp.gmail.com")
	// if err != nil {
	//     return err
	// }
	//
	// port, err := prompt.Int("Enter SMTP port", 587)
	// if err != nil {
	//     return err
	// }
	//
	// useTLS, err := prompt.Bool("Enable TLS", true)
	// if err != nil {
	//     return err
	// }
	//
	// fmt.Printf("Configuration: host=%s, port=%d, tls=%v\n", host, port, useTLS)
}
