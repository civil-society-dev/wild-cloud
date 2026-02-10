package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

var (
	devFollowLogs bool
	devNumLines   int
)

// devCmd represents the dev command group
var devCmd = &cobra.Command{
	Use:   "dev",
	Short: "Manage development services (API and web app)",
	Long:  `Start, stop, restart, and monitor the Wild Cloud development environment.`,
}

// startCmd starts the dev services
var startCmd = &cobra.Command{
	Use:   "start [api|web]",
	Short: "Start development services",
	Long: `Start development services in the background.

Examples:
  wild dev start       # Start both API and web app
  wild dev start api   # Start only API
  wild dev start web   # Start only web app`,
	ValidArgs: []string{"api", "web"},
	RunE: func(cmd *cobra.Command, args []string) error {
		projectRoot, err := getProjectRoot()
		if err != nil {
			return fmt.Errorf("failed to find project root: %w", err)
		}

		pidDir := filepath.Join(projectRoot, ".wild-pids")
		logDir := filepath.Join(projectRoot, ".wild-logs")

		// Create directories
		if err := os.MkdirAll(pidDir, 0755); err != nil {
			return fmt.Errorf("failed to create pid directory: %w", err)
		}
		if err := os.MkdirAll(logDir, 0755); err != nil {
			return fmt.Errorf("failed to create log directory: %w", err)
		}

		// Determine what to start
		startAPI := len(args) == 0 || args[0] == "api"
		startWeb := len(args) == 0 || args[0] == "web"

		if startAPI {
			if err := startAPIService(projectRoot, pidDir, logDir); err != nil {
				return err
			}
		}

		if startWeb {
			if err := startWebAppService(projectRoot, pidDir, logDir); err != nil {
				return err
			}
		}

		if len(args) == 0 {
			fmt.Println("\nBoth services running in background")
			fmt.Println("  • View logs: wild dev logs")
			fmt.Println("  • Check status: wild dev status")
			fmt.Println("  • Stop services: wild dev stop")
		}

		return nil
	},
}

// stopCmd stops the dev services
var stopCmd = &cobra.Command{
	Use:   "stop [api|web]",
	Short: "Stop development services",
	Long: `Stop development services.

Examples:
  wild dev stop       # Stop both API and web app
  wild dev stop api   # Stop only API
  wild dev stop web   # Stop only web app`,
	ValidArgs: []string{"api", "web"},
	RunE: func(cmd *cobra.Command, args []string) error {
		projectRoot, err := getProjectRoot()
		if err != nil {
			return fmt.Errorf("failed to find project root: %w", err)
		}

		pidDir := filepath.Join(projectRoot, ".wild-pids")

		// Determine what to stop
		stopAPI := len(args) == 0 || args[0] == "api"
		stopWeb := len(args) == 0 || args[0] == "web"

		stoppedAny := false

		if stopAPI {
			if err := stopService("API", filepath.Join(pidDir, "api.pid")); err == nil {
				stoppedAny = true
			}
		}

		if stopWeb {
			if err := stopService("Web app", filepath.Join(pidDir, "web.pid")); err == nil {
				stoppedAny = true
			}
		}

		if !stoppedAny {
			fmt.Println("No services running")
		}

		return nil
	},
}

// restartCmd restarts the dev services
var restartCmd = &cobra.Command{
	Use:   "restart [api|web]",
	Short: "Restart development services",
	Long: `Restart development services.

Examples:
  wild dev restart       # Restart both API and web app
  wild dev restart api   # Restart only API
  wild dev restart web   # Restart only web app`,
	ValidArgs: []string{"api", "web"},
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Restarting services...")

		// Stop services
		if err := stopCmd.RunE(cmd, args); err != nil {
			return err
		}

		// Brief pause
		time.Sleep(2 * time.Second)

		// Start services
		return startCmd.RunE(cmd, args)
	},
}

// statusCmd shows the status of dev services
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show status of development services",
	Long:  `Display the running status of API daemon and web app.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		projectRoot, err := getProjectRoot()
		if err != nil {
			return fmt.Errorf("failed to find project root: %w", err)
		}

		pidDir := filepath.Join(projectRoot, ".wild-pids")

		fmt.Println("Wild Cloud Development Status:")
		fmt.Println(strings.Repeat("-", 40))

		apiRunning, apiPid := isServiceRunning(filepath.Join(pidDir, "api.pid"))
		if apiRunning {
			fmt.Printf("API:     ✓ Running (PID %d)\n", apiPid)
			fmt.Println("         http://localhost:5055")
		} else {
			fmt.Println("API:     ✗ Not running")
		}

		webRunning, webPid := isServiceRunning(filepath.Join(pidDir, "web.pid"))
		if webRunning {
			fmt.Printf("Web App: ✓ Running (PID %d)\n", webPid)
			fmt.Println("         http://localhost:5173")
		} else {
			fmt.Println("Web App: ✗ Not running")
		}

		return nil
	},
}

// logsCmd shows logs from dev services
var logsCmd = &cobra.Command{
	Use:   "logs [api|web]",
	Short: "View logs from development services",
	Long: `Display logs from development services.

Examples:
  wild dev logs           # Show both API and web logs
  wild dev logs api       # Show only API logs
  wild dev logs web       # Show only web logs
  wild dev logs -f        # Follow all logs
  wild dev logs api -f    # Follow API logs
  wild dev logs -n 100    # Show last 100 lines`,
	ValidArgs: []string{"api", "web"},
	RunE: func(cmd *cobra.Command, args []string) error {
		projectRoot, err := getProjectRoot()
		if err != nil {
			return fmt.Errorf("failed to find project root: %w", err)
		}

		logDir := filepath.Join(projectRoot, ".wild-logs")

		// Determine what logs to show
		showAPI := len(args) == 0 || args[0] == "api"
		showWeb := len(args) == 0 || args[0] == "web"

		if len(args) > 0 {
			// Show specific service logs
			if args[0] == "api" {
				return showLogs(filepath.Join(logDir, "api.log"), devFollowLogs, devNumLines)
			} else if args[0] == "web" {
				return showLogs(filepath.Join(logDir, "web.log"), devFollowLogs, devNumLines)
			}
		}

		// Show both logs
		if showAPI {
			fmt.Println("=== API Logs ===")
			if err := showLogs(filepath.Join(logDir, "api.log"), false, devNumLines); err != nil {
				fmt.Printf("Error reading API logs: %v\n", err)
			}
		}

		if showWeb {
			fmt.Println("\n=== Web App Logs ===")
			if err := showLogs(filepath.Join(logDir, "web.log"), false, devNumLines); err != nil {
				fmt.Printf("Error reading web app logs: %v\n", err)
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(devCmd)
	devCmd.AddCommand(startCmd)
	devCmd.AddCommand(stopCmd)
	devCmd.AddCommand(restartCmd)
	devCmd.AddCommand(statusCmd)
	devCmd.AddCommand(logsCmd)

	// Flags for logs command
	logsCmd.Flags().BoolVarP(&devFollowLogs, "follow", "f", false, "Follow log output (like tail -f)")
	logsCmd.Flags().IntVarP(&devNumLines, "lines", "n", 50, "Number of lines to show")
}

// Helper functions

func getProjectRoot() (string, error) {
	// Use WILD_DEV_ROOT environment variable
	root := os.Getenv("WILD_DEV_ROOT")
	if root == "" {
		return "", fmt.Errorf("WILD_DEV_ROOT environment variable not set")
	}
	return root, nil
}

func startAPIService(projectRoot, pidDir, logDir string) error {
	pidFile := filepath.Join(pidDir, "api.pid")

	// Check if already running
	if running, pid := isServiceRunning(pidFile); running {
		fmt.Printf("API already running (PID %d)\n", pid)
		return nil
	}

	fmt.Println("Starting API...")

	apiDir := filepath.Join(projectRoot, "wild-cloud", "api")
	logFile := filepath.Join(logDir, "api.log")

	// Open log file
	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer f.Close()

	// Start API
	cmd := exec.Command("go", "run", ".")
	cmd.Dir = apiDir
	cmd.Stdout = f
	cmd.Stderr = f
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start API: %w", err)
	}

	// Write PID file
	if err := os.WriteFile(pidFile, []byte(fmt.Sprintf("%d", cmd.Process.Pid)), 0644); err != nil {
		cmd.Process.Kill()
		return fmt.Errorf("failed to write PID file: %w", err)
	}

	// Wait a moment and verify it started
	time.Sleep(2 * time.Second)
	if running, pid := isServiceRunning(pidFile); running {
		fmt.Printf("API started successfully (PID %d, logs: %s)\n", pid, logFile)
	} else {
		return fmt.Errorf("API failed to start")
	}

	return nil
}

func startWebAppService(projectRoot, pidDir, logDir string) error {
	pidFile := filepath.Join(pidDir, "web.pid")

	// Check if already running
	if running, pid := isServiceRunning(pidFile); running {
		fmt.Printf("Web app already running (PID %d)\n", pid)
		return nil
	}

	fmt.Println("Starting web app...")

	webDir := filepath.Join(projectRoot, "wild-cloud", "web")
	logFile := filepath.Join(logDir, "web.log")

	// Open log file
	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer f.Close()

	// Start web app
	cmd := exec.Command("pnpm", "run", "dev")
	cmd.Dir = webDir
	cmd.Stdout = f
	cmd.Stderr = f
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start web app: %w", err)
	}

	// Write PID file
	if err := os.WriteFile(pidFile, []byte(fmt.Sprintf("%d", cmd.Process.Pid)), 0644); err != nil {
		cmd.Process.Kill()
		return fmt.Errorf("failed to write PID file: %w", err)
	}

	// Wait a moment and verify it started
	time.Sleep(3 * time.Second)
	if running, pid := isServiceRunning(pidFile); running {
		fmt.Printf("Web app started successfully (PID %d, logs: %s)\n", pid, logFile)
	} else {
		return fmt.Errorf("Web app failed to start")
	}

	return nil
}

func stopService(name, pidFile string) error {
	// Try to stop process from PID file first
	running, pid := isServiceRunning(pidFile)
	if running {
		return stopProcess(name, pid, pidFile)
	}

	// PID file doesn't exist or process is dead, try to find running processes
	var pids []int
	if name == "API" {
		pids = findAPIProcesses()
	} else if name == "Web app" {
		pids = findWebProcesses()
	}

	if len(pids) == 0 {
		fmt.Printf("%s not running\n", name)
		return fmt.Errorf("not running")
	}

	// Stop all found processes
	for _, pid := range pids {
		if err := stopProcess(name, pid, pidFile); err != nil {
			fmt.Printf("Warning: Failed to stop %s (PID %d): %v\n", name, pid, err)
		}
	}

	return nil
}

func stopProcess(name string, pid int, pidFile string) error {
	fmt.Printf("Stopping %s (PID %d)...\n", name, pid)

	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("failed to find process: %w", err)
	}

	// Try graceful shutdown first
	if err := process.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("failed to send SIGTERM: %w", err)
	}

	// Wait for graceful shutdown
	for i := 0; i < 10; i++ {
		time.Sleep(500 * time.Millisecond)
		if err := process.Signal(syscall.Signal(0)); err != nil {
			// Process is dead
			os.Remove(pidFile)
			fmt.Printf("%s stopped successfully\n", name)
			return nil
		}
	}

	// Force kill if still running
	fmt.Printf("%s did not stop gracefully, force killing...\n", name)
	if err := process.Kill(); err != nil {
		return fmt.Errorf("failed to kill process: %w", err)
	}

	os.Remove(pidFile)
	fmt.Printf("%s force killed\n", name)
	return nil
}

func findAPIProcesses() []int {
	var pids []int

	// Look for:
	// 1. wildd binary
	// 2. go run in api directory
	cmd := exec.Command("pgrep", "-f", "wildd|wild-cloud/api")
	output, err := cmd.Output()
	if err != nil {
		return pids
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		if pid, err := strconv.Atoi(line); err == nil {
			pids = append(pids, pid)
		}
	}

	return pids
}

func findWebProcesses() []int {
	var pids []int

	// Look for vite dev server (pnpm run dev in web directory)
	cmd := exec.Command("pgrep", "-f", "vite.*wild-cloud/web")
	output, err := cmd.Output()
	if err != nil {
		return pids
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		if pid, err := strconv.Atoi(line); err == nil {
			pids = append(pids, pid)
		}
	}

	return pids
}

func isServiceRunning(pidFile string) (bool, int) {
	data, err := os.ReadFile(pidFile)
	if err != nil {
		return false, 0
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return false, 0
	}

	// Check if process exists
	process, err := os.FindProcess(pid)
	if err != nil {
		return false, 0
	}

	// Send signal 0 to check if process is alive
	if err := process.Signal(syscall.Signal(0)); err != nil {
		return false, 0
	}

	return true, pid
}

func showLogs(logFile string, follow bool, lines int) error {
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		fmt.Printf("No logs found at %s\n", logFile)
		return nil
	}

	if follow {
		// Use tail -f
		cmd := exec.Command("tail", "-f", logFile)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}

	// Show last N lines
	cmd := exec.Command("tail", "-n", strconv.Itoa(lines), logFile)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
