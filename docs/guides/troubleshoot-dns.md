# Troubleshoot DNS

If DNS resolution isn't working properly:

1. Check CoreDNS status:
   ```bash
   kubectl get pods -n kube-system -l k8s-app=kube-dns
   kubectl logs -l k8s-app=kube-dns -n kube-system
   ```

2. Verify CoreDNS configuration:
   ```bash
   kubectl get configmap -n kube-system coredns -o yaml
   ```

3. Test DNS resolution from inside the cluster:
   ```bash
   kubectl run -i --tty --rm debug --image=busybox --restart=Never -- nslookup kubernetes.default
   ```

