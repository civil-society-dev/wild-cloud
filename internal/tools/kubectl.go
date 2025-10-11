package tools

import (
	"os/exec"
)

// Kubectl provides a thin wrapper around the kubectl command-line tool
type Kubectl struct {
	kubeconfigPath string
}

// NewKubectl creates a new Kubectl wrapper
func NewKubectl(kubeconfigPath string) *Kubectl {
	return &Kubectl{
		kubeconfigPath: kubeconfigPath,
	}
}

// DeploymentExists checks if a deployment exists in the specified namespace
func (k *Kubectl) DeploymentExists(name, namespace string) bool {
	args := []string{
		"get", "deployment", name,
		"-n", namespace,
	}

	if k.kubeconfigPath != "" {
		args = append([]string{"--kubeconfig", k.kubeconfigPath}, args...)
	}

	cmd := exec.Command("kubectl", args...)
	err := cmd.Run()
	return err == nil
}
