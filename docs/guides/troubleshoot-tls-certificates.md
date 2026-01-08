# Troubleshoot TLS Certificates

If services show invalid certificates:

1. Check certificate status:
   ```bash
   kubectl get certificates -A
   ```

2. Examine certificate details:
   ```bash
   kubectl describe certificate <cert-name> -n <namespace>
   ```

3. Check for cert-manager issues:
   ```bash
   kubectl get pods -n cert-manager
   kubectl logs -l app=cert-manager -n cert-manager
   ```

4. Verify the Cloudflare API token is correctly set up:
   ```bash
   kubectl get secret cloudflare-api-token -n internal
   ```
