package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"

	v1 "github.com/wild-cloud/wild-central/daemon/internal/api/v1"
)

var startTime time.Time

func main() {
	// Record start time
	startTime = time.Now()

	// Get data directory from environment or use default
	dataDir := os.Getenv("WILD_CENTRAL_DATA")
	if dataDir == "" {
		dataDir = "/var/lib/wild-central"
	}

	// Get directory path from environment (required)
	directoryPath := os.Getenv("WILD_DIRECTORY")
	if directoryPath == "" {
		log.Fatal("WILD_DIRECTORY environment variable is required")
	}

	// Create API handler with all dependencies
	api, err := v1.NewAPI(dataDir, directoryPath)
	if err != nil {
		log.Fatalf("Failed to initialize API: %v", err)
	}

	// Set up HTTP router
	router := mux.NewRouter()

	// Register Phase 1 API routes
	api.RegisterRoutes(router)

	// Health check endpoint
	router.HandleFunc("/api/v1/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"ok"}`)
	}).Methods("GET")

	// Status endpoint
	router.HandleFunc("/api/v1/status", func(w http.ResponseWriter, r *http.Request) {
		api.StatusHandler(w, r, startTime, dataDir, directoryPath)
	}).Methods("GET")

	// Default server settings
	host := "0.0.0.0"
	port := 5055

	addr := fmt.Sprintf("%s:%d", host, port)
	log.Printf("Starting wild-central daemon on %s", addr)
	log.Printf("Data directory: %s", dataDir)
	log.Printf("Wild Cloud Directory: %s", directoryPath)

	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}
