# Troubleshoot Wild Cloud Cluster issues

## General Troubleshooting Steps

1. **Check Node Status**:
   ```bash
   kubectl get nodes
   kubectl describe node <node-name>
   ```

1. **Check Component Status**:
   ```bash
   # Check all pods across all namespaces
   kubectl get pods -A
   
   # Look for pods that aren't Running or Ready
   kubectl get pods -A | grep -v "Running\|Completed"
   ```

