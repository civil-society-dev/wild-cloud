package v1

import (
	"encoding/json"
	"net/http"
	"os"
	"os/exec"

	"github.com/creack/pty"
	"github.com/gorilla/websocket"

	"github.com/wild-cloud/wild-central/daemon/internal/tools"
)

// terminalResize represents a resize message from the client
type terminalResize struct {
	Type string `json:"type"`
	Cols int    `json:"cols"`
	Rows int    `json:"rows"`
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins in development
		// TODO: Configure properly for production
		return true
	},
}

// TerminalWebSocket handles WebSocket connections for interactive terminal sessions
func (api *API) TerminalWebSocket(w http.ResponseWriter, r *http.Request) {
	instanceName := GetInstanceName(r)

	// Upgrade to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	// Get instance paths
	instancePath := tools.GetInstancePath(api.dataDir, instanceName)
	kubeconfigPath := tools.GetKubeconfigPath(api.dataDir, instanceName)
	talosconfigPath := tools.GetTalosconfigPath(api.dataDir, instanceName)

	// Create shell with PTY
	// Use bash for readline support (history, arrow keys, etc.)
	cmd := exec.Command("/bin/bash")
	cmd.Dir = instancePath
	// Store bash history per-instance
	historyFile := instancePath + "/.bash_history"

	cmd.Env = append(os.Environ(),
		"KUBECONFIG="+kubeconfigPath,
		"TALOSCONFIG="+talosconfigPath,
		"WILD_INSTANCE="+instanceName,
		"WILD_DATA_DIR="+api.dataDir,
		"WILD_INSTANCE_DIR="+instancePath,
		"HISTFILE="+historyFile,
		"PROMPT_COMMAND=history -a", // Save history after each command
		"TERM=xterm-256color",
	)

	ptmx, err := pty.Start(cmd)
	if err != nil {
		conn.WriteMessage(websocket.TextMessage, []byte("Failed to start shell: "+err.Error()))
		return
	}
	defer ptmx.Close()
	defer cmd.Process.Kill()

	// Channel to signal when to stop
	done := make(chan struct{})

	// Goroutine: PTY -> WebSocket
	go func() {
		buf := make([]byte, 4096)
		for {
			select {
			case <-done:
				return
			default:
				n, err := ptmx.Read(buf)
				if err != nil {
					return
				}
				if err := conn.WriteMessage(websocket.BinaryMessage, buf[:n]); err != nil {
					return
				}
			}
		}
	}()

	// Main loop: WebSocket -> PTY
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			close(done)
			return
		}

		// Check if this is a resize message
		var resize terminalResize
		if err := json.Unmarshal(msg, &resize); err == nil && resize.Type == "resize" {
			if resize.Cols > 0 && resize.Rows > 0 {
				pty.Setsize(ptmx, &pty.Winsize{
					Cols: uint16(resize.Cols),
					Rows: uint16(resize.Rows),
				})
			}
			continue
		}

		// Regular input - write to PTY
		if _, err := ptmx.Write(msg); err != nil {
			close(done)
			return
		}
	}
}
