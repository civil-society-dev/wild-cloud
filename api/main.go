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
		api.StatusHandler(w, r, startTime, dataDir, appsDir)
	}).Methods("GET")

	// Configure CORS
	// Default to development origins
	allowedOrigins := []string{
		"http://localhost:5173", // Vite dev server
		"http://localhost:5174", // Alternative port
		"http://localhost:3000", // Common React dev port
		"http://127.0.0.1:5173",
		"http://127.0.0.1:5174",
		"http://127.0.0.1:3000",
		"http://wild-central:5173",
		"http://wild-central.lan:5173",
	}

	// Override with production origins if set
	if corsOrigins := os.Getenv("WILD_CORS_ORIGINS"); corsOrigins != "" {
		// Split comma-separated origins
		allowedOrigins = splitAndTrim(corsOrigins, ",")
		log.Printf("CORS configured for production origins: %v", allowedOrigins)
	} else {
		log.Printf("CORS configured for development origins")
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
	log.Printf("CORS enabled for development origins")

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
