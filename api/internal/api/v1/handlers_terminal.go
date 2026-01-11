package v1

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"os/exec"

	"github.com/wild-cloud/wild-central/daemon/internal/tools"
)

// TerminalExecRequest represents a command execution request
type TerminalExecRequest struct {
	Command string `json:"command"`
}

// TerminalExecResponse represents the result of command execution
type TerminalExecResponse struct {
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode int    `json:"exit_code"`
}

// TerminalExec executes a shell command on Wild Central with instance context
func (api *API) TerminalExec(w http.ResponseWriter, r *http.Request) {
	instanceName := GetInstanceName(r)

	var req TerminalExecRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Command == "" {
		respondError(w, http.StatusBadRequest, "Command is required")
		return
	}

	// Get instance-specific paths
	instancePath := tools.GetInstancePath(api.dataDir, instanceName)
	kubeconfigPath := tools.GetKubeconfigPath(api.dataDir, instanceName)
	talosconfigPath := tools.GetTalosconfigPath(api.dataDir, instanceName)
	configPath := tools.GetInstanceConfigPath(api.dataDir, instanceName)

	// Get VIP for talosctl nodes
	vip, _ := api.config.GetConfigValue(configPath, "cluster.nodes.control.vip")

	cmd := exec.Command("/bin/sh", "-c", req.Command)

	// Set working directory to instance path
	cmd.Dir = instancePath

	// Set environment variables like the CLI would
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, "KUBECONFIG="+kubeconfigPath)
	cmd.Env = append(cmd.Env, "TALOSCONFIG="+talosconfigPath)
	if vip != "" {
		cmd.Env = append(cmd.Env, "TALOSCTL_NODES="+vip)
	}
	cmd.Env = append(cmd.Env, "WILD_INSTANCE="+instanceName)
	cmd.Env = append(cmd.Env, "WILD_DATA_DIR="+api.dataDir)
	cmd.Env = append(cmd.Env, "WILD_INSTANCE_DIR="+instancePath)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitCode := 0
	if exitErr, ok := err.(*exec.ExitError); ok {
		exitCode = exitErr.ExitCode()
	} else if err != nil {
		exitCode = 1
		stderr.WriteString(err.Error())
	}

	respondJSON(w, http.StatusOK, TerminalExecResponse{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: exitCode,
	})
}
