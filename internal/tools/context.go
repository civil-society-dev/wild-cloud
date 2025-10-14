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

// GetInstancePath returns the path to an instance directory
func GetInstancePath(dataDir, instanceName string) string {
	return filepath.Join(dataDir, "instances", instanceName)
}

// GetInstanceConfigPath returns the path to an instance's config file
func GetInstanceConfigPath(dataDir, instanceName string) string {
	return filepath.Join(dataDir, "instances", instanceName, "config.yaml")
}

// GetInstanceSecretsPath returns the path to an instance's secrets file
func GetInstanceSecretsPath(dataDir, instanceName string) string {
	return filepath.Join(dataDir, "instances", instanceName, "secrets.yaml")
}

// GetInstanceTalosPath returns the path to an instance's talos directory
func GetInstanceTalosPath(dataDir, instanceName string) string {
	return filepath.Join(dataDir, "instances", instanceName, "talos")
}

// GetInstancePXEPath returns the path to an instance's PXE directory
func GetInstancePXEPath(dataDir, instanceName string) string {
	return filepath.Join(dataDir, "instances", instanceName, "pxe")
}

// GetInstanceOperationsPath returns the path to an instance's operations directory
func GetInstanceOperationsPath(dataDir, instanceName string) string {
	return filepath.Join(dataDir, "instances", instanceName, "operations")
}

// GetInstanceBackupsPath returns the path to an instance's backups directory
func GetInstanceBackupsPath(dataDir, instanceName string) string {
	return filepath.Join(dataDir, "instances", instanceName, "backups")
}

// GetInstanceDiscoveryPath returns the path to an instance's discovery directory
func GetInstanceDiscoveryPath(dataDir, instanceName string) string {
	return filepath.Join(dataDir, "instances", instanceName, "discovery")
}

// GetInstancesPath returns the path to the instances directory
func GetInstancesPath(dataDir string) string {
	return filepath.Join(dataDir, "instances")
}

// GetInstancesLockPath returns the path to the instances directory lock file
func GetInstancesLockPath(dataDir string) string {
	return filepath.Join(dataDir, "instances", ".lock")
}
