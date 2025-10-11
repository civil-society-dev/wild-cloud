# NVIDIA Device Plugin

The NVIDIA Device Plugin for Kubernetes enables GPU scheduling and resource management on nodes with NVIDIA GPUs.

## Overview

This service deploys the official NVIDIA Device Plugin as a DaemonSet that:
- Discovers NVIDIA GPUs on worker nodes
- Labels nodes with GPU product information (e.g., `nvidia.com/gpu.product=GeForce-RTX-4090`)
- Advertises GPU resources (`nvidia.com/gpu`) to the Kubernetes scheduler
- Enables pods to request GPU resources

## Prerequisites

Before installing the NVIDIA Device Plugin, ensure that:

1. **NVIDIA Drivers** are installed (>= 384.81)
2. **nvidia-container-toolkit** is installed (>= 1.7.0)
3. **nvidia-container-runtime** is configured as the default container runtime
4. Worker nodes have NVIDIA GPUs

### Talos Linux Requirements

For Talos Linux nodes, you need:
- NVIDIA drivers extension in the Talos schematic
- nvidia-container-toolkit extension
- Proper container runtime configuration

## Installation

```bash
# Configure and install the service
wild-cluster-services-configure nvidia-device-plugin
wild-cluster-install nvidia-device-plugin
```

## Verification

After installation, verify the plugin is working:

```bash
# Check plugin pods are running
kubectl get pods -n kube-system | grep nvidia

# Verify GPU resources are advertised
kubectl get nodes -o json | jq '.items[].status.capacity | select(has("nvidia.com/gpu"))'

# Check GPU node labels
kubectl get nodes --show-labels | grep nvidia
```

## Usage in Applications

Once installed, applications can request GPU resources:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: gpu-app
spec:
  template:
    spec:
      containers:
      - name: app
        image: nvidia/cuda:latest
        resources:
          requests:
            nvidia.com/gpu: 1
          limits:
            nvidia.com/gpu: 1
```

## Troubleshooting

### Plugin Not Starting
- Verify NVIDIA drivers are installed on worker nodes
- Check that nvidia-container-toolkit is properly configured
- Ensure worker nodes are not tainted in a way that prevents scheduling

### No GPU Resources Advertised
- Check plugin logs: `kubectl logs -n kube-system -l name=nvidia-device-plugin-ds`
- Verify NVIDIA runtime is the default container runtime
- Ensure GPUs are detected by the driver: check node logs for GPU detection messages

## Configuration

The plugin uses the following configuration:
- **Image**: `nvcr.io/nvidia/k8s-device-plugin:v0.17.1`
- **Namespace**: `kube-system`
- **Priority Class**: `system-node-critical`
- **Tolerations**: Schedules on nodes with `nvidia.com/gpu` taint

## References

- [Official NVIDIA Device Plugin Repository](https://github.com/NVIDIA/k8s-device-plugin)
- [Kubernetes GPU Scheduling Documentation](https://kubernetes.io/docs/tasks/manage-gpus/scheduling-gpus/)
- [NVIDIA Container Toolkit Documentation](https://docs.nvidia.com/datacenter/cloud-native/container-toolkit/)