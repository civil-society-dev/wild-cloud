package cmd

import (
	"fmt"
	"time"

	"github.com/r3labs/sse/v2"
	"github.com/spf13/cobra"

	"github.com/wild-cloud/wild-central/wild/internal/config"
)

// Operation commands
var operationCmd = &cobra.Command{
	Use:   "operation",
	Short: "Manage operations",
}

var operationGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get operation status",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		inst, err := getInstanceName()
		if err != nil {
			return err
		}

		resp, err := apiClient.Get(fmt.Sprintf("/api/v1/operations/%s?instance=%s", args[0], inst))
		if err != nil {
			return err
		}

		if outputFormat == "json" {
			return printJSON(resp.Data)
		}

		return printYAML(resp.Data)
	},
}

var operationListCmd = &cobra.Command{
	Use:   "list",
	Short: "List operations",
	RunE: func(cmd *cobra.Command, args []string) error {
		resp, err := apiClient.Get("/api/v1/operations")
		if err != nil {
			return err
		}

		if outputFormat == "json" {
			return printJSON(resp.Data)
		}

		ops := resp.GetArray("operations")
		if len(ops) == 0 {
			fmt.Println("No operations found")
			return nil
		}

		fmt.Printf("%-10s  %-12s  %-12s  %-10s\n", "ID", "TYPE", "STATUS", "PROGRESS")
		fmt.Println("--------------------------------------------------------")
		for _, op := range ops {
			if m, ok := op.(map[string]interface{}); ok {
				fmt.Printf("%-10s  %-12s  %-12s  %d%%\n",
					m["id"], m["type"], m["status"], int(m["progress"].(float64)))
			}
		}
		return nil
	},
}

func init() {
	operationCmd.AddCommand(operationGetCmd)
	operationCmd.AddCommand(operationListCmd)
}

// streamOperationOutput streams operation output via SSE
func streamOperationOutput(opID string) error {
	// Get instance name
	inst, err := getInstanceName()
	if err != nil {
		return err
	}

	// Get base URL
	baseURL := daemonURL
	if baseURL == "" {
		baseURL = config.GetDaemonURL()
	}

	// Connect to SSE stream
	url := fmt.Sprintf("%s/api/v1/operations/%s/stream?instance=%s", baseURL, opID, inst)
	client := sse.NewClient(url)
	events := make(chan *sse.Event)

	err = client.SubscribeChan("messages", events)
	if err != nil {
		return fmt.Errorf("failed to subscribe to SSE: %w", err)
	}

	// Poll for completion in background
	done := make(chan bool, 1)
	go func() {
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		for range ticker.C {
			resp, err := apiClient.Get(fmt.Sprintf("/api/v1/operations/%s?instance=%s", opID, inst))
			if err == nil {
				status := resp.GetString("status")
				if status == "completed" || status == "failed" {
					time.Sleep(500 * time.Millisecond) // Give SSE time to flush
					done <- true
					return
				}
			}
		}
	}()

	// Stream events
	for {
		select {
		case msg, ok := <-events:
			if !ok {
				// Channel closed
				return nil
			}
			if msg != nil {
				// Check for completion event
				if string(msg.Event) == "complete" {
					return nil
				}
				// Print data with newline
				if msg.Data != nil {
					fmt.Println(string(msg.Data))
				}
			}
		case <-done:
			return nil
		}
	}
}
