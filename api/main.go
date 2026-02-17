package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/cors"

	v1 "github.com/wild-cloud/wild-central/daemon/internal/api/v1"
	"github.com/wild-cloud/wild-central/daemon/internal/backup"
)

var startTime time.Time

// splitAndTrim splits a string by delimiter and trims whitespace from each part
func splitAndTrim(s string, sep string) []string {
	parts := strings.Split(s, sep)
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func main() {
	// Record start time
	startTime = time.Now()

	// Get data directory from environment or use default
	dataDir := os.Getenv("WILD_API_DATA_DIR")
	if dataDir == "" {
		dataDir = "/var/lib/wild-central"
	}

	// Get apps directory from environment or use default
	// Note: Setup files (cluster-services, cluster-nodes, etc.) are now embedded in the binary
	appsDir := os.Getenv("WILD_DIRECTORY")
	if appsDir == "" {
		// Default apps directory
		appsDir = "/opt/wild-cloud/apps"
		log.Printf("WILD_DIRECTORY not set, using default apps directory: %s", appsDir)
	} else {
		// If WILD_DIRECTORY is set, use it as-is for backward compatibility
		// (it might point to the old directory structure with apps/ subdirectory)
		log.Printf("Using WILD_DIRECTORY for apps: %s", appsDir)
	}

	// Create API handler with all dependencies
	api, err := v1.NewAPI(dataDir, appsDir)
	if err != nil {
		log.Fatalf("Failed to initialize API: %v", err)
	}

	// Initialize backup scheduler
	backupMgr := backup.NewManager(dataDir)
	scheduler := backup.NewScheduler(backupMgr)
	api.SetBackupScheduler(scheduler)
	scheduler.Start()
	log.Println("Backup scheduler initialized")

	// Start central status SSE broadcaster
	api.StartCentralStatusBroadcaster(startTime)
	log.Println("Central status broadcaster started")

	// Set up HTTP router
	router := mux.NewRouter()
	api.RegisterRoutes(router)

	// Health check endpoint
	router.HandleFunc("/api/v1/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"ok"}`)
	}).Methods("GET")

	// Status endpoint
	router.HandleFunc("/api/v1/status", func(w http.ResponseWriter, r *http.Request) {
		api.StatusHandler(w, r, startTime, dataDir, appsDir)
	}).Methods("GET")

	// Configure CORS
	var allowedOrigins []string

	if corsOrigins := os.Getenv("WILD_CORS_ORIGINS"); corsOrigins != "" {
		// Use explicitly configured origins
		allowedOrigins = splitAndTrim(corsOrigins, ",")
		log.Printf("CORS configured with explicit origins: %v", allowedOrigins)
	} else {
		// Auto-detect origins based on hostname
		allowedOrigins = []string{
			"http://localhost",
			"http://localhost:80",
			"http://127.0.0.1",
			"http://127.0.0.1:80",
		}

		// Add machine hostname (for production access via nginx)
		if hostname, err := os.Hostname(); err == nil && hostname != "" {
			allowedOrigins = append(allowedOrigins,
				fmt.Sprintf("http://%s", hostname),
				fmt.Sprintf("http://%s:80", hostname),
				fmt.Sprintf("http://%s.local", hostname),
				fmt.Sprintf("http://%s.lan", hostname),
				fmt.Sprintf("http://%s.local:5173", hostname),
				fmt.Sprintf("http://%s.lan:5173", hostname),
				fmt.Sprintf("http://%s.local:5174", hostname),
				fmt.Sprintf("http://%s.lan:5174", hostname),
				// Add development server ports for hostname
				fmt.Sprintf("http://%s:5173", hostname),
				fmt.Sprintf("http://%s:5174", hostname),
			)
			log.Printf("Added hostname-based CORS origins for: %s", hostname)
		}

		// Add development server ports
		allowedOrigins = append(allowedOrigins,
			"http://localhost:5173",
			"http://localhost:5174",
			"http://localhost:3000",
			"http://127.0.0.1:5173",
			"http://127.0.0.1:5174",
			"http://127.0.0.1:3000",
		)

		log.Printf("CORS configured with auto-detected origins: %v", allowedOrigins)
	}

	corsHandler := cors.New(cors.Options{
		AllowedOrigins: allowedOrigins,
		AllowedMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
			http.MethodOptions,
		},
		AllowedHeaders: []string{
			"Accept",
			"Authorization",
			"Content-Type",
			"X-CSRF-Token",
		},
		ExposedHeaders: []string{
			"Link",
		},
		AllowCredentials: true,
		MaxAge:           300, // 5 minutes
	})

	// Wrap router with CORS middleware
	handler := corsHandler.Handler(router)

	// Default server settings
	host := "0.0.0.0"
	port := 5055

	addr := fmt.Sprintf("%s:%d", host, port)
	log.Printf("Starting wild-central daemon on %s", addr)
	log.Printf("Data directory: %s", dataDir)
	log.Printf("Apps directory: %s", appsDir)

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start HTTP server in goroutine
	go func() {
		if err := http.ListenAndServe(addr, handler); err != nil {
			log.Fatal("Server failed to start:", err)
		}
	}()

	// Wait for shutdown signal
	<-sigChan
	log.Println("Shutting down gracefully...")
	scheduler.Stop()
	log.Println("Shutdown complete")
}
