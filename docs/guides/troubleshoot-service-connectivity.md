# Troubleshoot Service Connectivity

If services can't communicate:

1. Check network policies:
   ```bash
   kubectl get networkpolicies -A
   ```

2. Verify service endpoints:
   ```bash
   kubectl get endpoints -n <namespace>
   ```

3. Test connectivity from within the cluster:
   ```bash
   kubectl run -i --tty --rm debug --image=busybox --restart=Never -- wget -O- <service-name>.<namespace>
   ```
