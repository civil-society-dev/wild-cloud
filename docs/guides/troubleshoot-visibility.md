# Troubleshoot Service Visibility

This guide covers common issues with accessing services from outside the cluster and how to diagnose and fix them.

## Common Issues

External access to your services might fail for several reasons:

1. **DNS Resolution Issues** - Domain names not resolving to the correct IP address
2. **Network Connectivity Issues** - Traffic can't reach the cluster's external IP
3. **TLS Certificate Issues** - Invalid or missing certificates
4. **Ingress/Service Configuration Issues** - Incorrectly configured routing

## Diagnostic Steps

### 1. Check DNS Resolution

**Symptoms:**

- Browser shows "site cannot be reached" or "server IP address could not be found"
- `ping` or `nslookup` commands fail for your domain
- Your service DNS records don't appear in CloudFlare or your DNS provider

**Checks:**

```bash
# Check if your domain resolves (from outside the cluster)
nslookup yourservice.yourdomain.com

# Check if ExternalDNS is running
kubectl get pods -n externaldns

# Check ExternalDNS logs for errors
kubectl logs -n externaldns -l app=external-dns  < /dev/null |  grep -i error
kubectl logs -n externaldns -l app=external-dns | grep -i "your-service-name"

# Check if CloudFlare API token is configured correctly
kubectl get secret cloudflare-api-token -n externaldns
```

**Common Issues:**

a) **ExternalDNS Not Running**: The ExternalDNS pod is not running or has errors.

b) **Cloudflare API Token Issues**: The API token is invalid, expired, or doesn't have the right permissions.

c) **Domain Filter Mismatch**: ExternalDNS is configured with a `--domain-filter` that doesn't match your domain.

d) **Annotations Missing**: Service or Ingress is missing the required ExternalDNS annotations.

**Solutions:**

```bash
# 1. Recreate CloudFlare API token secret
kubectl create secret generic cloudflare-api-token \
  --namespace externaldns \
  --from-literal=api-token="your-api-token" \
  --dry-run=client -o yaml | kubectl apply -f -

# 2. Check and set proper annotations on your Ingress:
kubectl annotate ingress your-ingress -n your-namespace \
  external-dns.alpha.kubernetes.io/hostname=your-service.your-domain.com

# 3. Restart ExternalDNS
kubectl rollout restart deployment -n externaldns external-dns
```

### 2. Check Network Connectivity

**Symptoms:**

- DNS resolves to the correct IP but the service is still unreachable
- Only some services are unreachable while others work
- Network timeout errors

**Checks:**

```bash
# Check if MetalLB is running
kubectl get pods -n metallb-system

# Check MetalLB IP address pool
kubectl get ipaddresspools.metallb.io -n metallb-system

# Verify the service has an external IP
kubectl get svc -n your-namespace your-service
```

**Common Issues:**

a) **MetalLB Configuration**: The IP pool doesn't match your network or is exhausted.

b) **Firewall Issues**: Firewall is blocking traffic to your cluster's external IP.

c) **Router Configuration**: NAT or port forwarding issues if using a router.

**Solutions:**

```bash
# 1. Check and update MetalLB configuration
kubectl apply -f infrastructure_setup/metallb/metallb-pool.yaml

# 2. Check service external IP assignment
kubectl describe svc -n your-namespace your-service
```

### 3. Check TLS Certificates

**Symptoms:**

- Browser shows certificate errors
- "Your connection is not private" warnings
- Cert-manager logs show errors

**Checks:**

```bash
# Check certificate status
kubectl get certificates -A

# Check cert-manager logs
kubectl logs -n cert-manager -l app=cert-manager

# Check if your ingress is using the correct certificate
kubectl get ingress -n your-namespace your-ingress -o yaml
```

**Common Issues:**

a) **Certificate Issuance Failures**: DNS validation or HTTP validation failing.

b) **Wrong Secret Referenced**: Ingress is referencing a non-existent certificate secret.

c) **Expired Certificate**: Certificate has expired and wasn't renewed.

**Solutions:**

```bash
# 1. Check and recreate certificates
kubectl apply -f infrastructure_setup/cert-manager/wildcard-certificate.yaml

# 2. Update ingress to use correct secret
kubectl patch ingress your-ingress -n your-namespace --type=json \
  -p='[{"op": "replace", "path": "/spec/tls/0/secretName", "value": "correct-secret-name"}]'
```

### 4. Check Ingress Configuration

**Symptoms:**

- HTTP 404, 503, or other error codes
- Service accessible from inside cluster but not outside
- Traffic routed to wrong service

**Checks:**

```bash
# Check ingress status
kubectl get ingress -n your-namespace

# Check Traefik logs
kubectl logs -n kube-system -l app.kubernetes.io/name=traefik

# Check ingress configuration
kubectl describe ingress -n your-namespace your-ingress
```

**Common Issues:**

a) **Incorrect Service Targeting**: Ingress is pointing to wrong service or port.

b) **Traefik Configuration**: IngressClass or middleware issues.

c) **Path Configuration**: Incorrect path prefixes or regex.

**Solutions:**

```bash
# 1. Verify ingress configuration
kubectl edit ingress -n your-namespace your-ingress

# 2. Check that the referenced service exists
kubectl get svc -n your-namespace

# 3. Restart Traefik if needed
kubectl rollout restart deployment -n kube-system traefik
```

## Advanced Diagnostics

For more complex issues, you can use port-forwarding to test services directly:

```bash
# Port-forward the service directly
kubectl port-forward -n your-namespace svc/your-service 8080:80

# Then test locally
curl http://localhost:8080
```

You can also deploy a debug pod to test connectivity from inside the cluster:

```bash
# Start a debug pod
kubectl run -i --tty --rm debug --image=busybox --restart=Never -- sh

# Inside the pod, test DNS and connectivity
nslookup your-service.your-namespace.svc.cluster.local
wget -O- http://your-service.your-namespace.svc.cluster.local
```

## ExternalDNS Specifics

ExternalDNS can be particularly troublesome. Here are specific debugging steps:

1. **Check Log Level**: Set `--log-level=debug` for more detailed logs
2. **Check Domain Filter**: Ensure `--domain-filter` includes your domain
3. **Check Provider**: Ensure `--provider=cloudflare` (or your DNS provider)
4. **Verify API Permissions**: CloudFlare token needs Zone.Zone and Zone.DNS permissions
5. **Check TXT Records**: ExternalDNS uses TXT records for ownership tracking

```bash
# Restart with verbose logging
kubectl set env deployment/external-dns -n externaldns -- --log-level=debug

# Check for specific domain errors
kubectl logs -n externaldns -l app=external-dns | grep -i yourservice.yourdomain.com
```

## CloudFlare Specific Issues

When using CloudFlare, additional issues may arise:

1. **API Rate Limiting**: CloudFlare may rate limit frequent API calls
2. **DNS Propagation**: Changes may take time to propagate through CloudFlare's CDN
3. **Proxied Records**: The `external-dns.alpha.kubernetes.io/cloudflare-proxied` annotation controls whether CloudFlare proxies traffic
4. **Access Restrictions**: CloudFlare Access or Page Rules may restrict access
5. **API Token Permissions**: The token must have Zone:Zone:Read and Zone:DNS:Edit permissions
6. **Zone Detection**: If using subdomains, ensure the parent domain is included in the domain filter

Check CloudFlare dashboard for:

- DNS record existence
- API access logs
- DNS settings including proxy status
- Any error messages or rate limit warnings
