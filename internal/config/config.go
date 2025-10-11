// Package config handles CLI configuration
package config

import (
	"os"
)

// GetDaemonURL returns the daemon URL from environment or default
func GetDaemonURL() string {
	// Check environment variable first
	if url := os.Getenv("WILD_DAEMON_URL"); url != "" {
		return url
	}

	// Use default matching daemon's port
	return "http://localhost:5055"
}
