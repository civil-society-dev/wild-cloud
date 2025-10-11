package tools

import (
	"os"
	"os/exec"
	"path/filepath"
)

// WithTalosconfig sets the TALOSCONFIG environment variable for a command
// This allows talosctl commands to use the correct context without global state
func WithTalosconfig(cmd *exec.Cmd, talosconfigPath string) *exec.Cmd {
	if cmd.Env == nil {
		cmd.Env = os.Environ()
	}
	cmd.Env = append(cmd.Env, "TALOSCONFIG="+talosconfigPath)
	return cmd
}

// WithKubeconfig sets the KUBECONFIG environment variable for a command
// This allows kubectl commands to use the correct context without global state
func WithKubeconfig(cmd *exec.Cmd, kubeconfigPath string) *exec.Cmd {
	if cmd.Env == nil {
		cmd.Env = os.Environ()
	}
	cmd.Env = append(cmd.Env, "KUBECONFIG="+kubeconfigPath)
	return cmd
}

// GetTalosconfigPath returns the path to the talosconfig for an instance
func GetTalosconfigPath(dataDir, instanceName string) string {
	return filepath.Join(dataDir, "instances", instanceName, "talos", "generated", "talosconfig")
}

// GetKubeconfigPath returns the path to the kubeconfig for an instance
func GetKubeconfigPath(dataDir, instanceName string) string {
	return filepath.Join(dataDir, "instances", instanceName, "kubeconfig")
}
