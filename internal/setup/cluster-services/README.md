# Wild Cloud Cluster Services

Creates a fully functional personal cloud infrastructure on a bare metal Kubernetes cluster that provides:

1. **External access** to services via configured domain names (using ${DOMAIN})
2. **Internal-only access** to admin interfaces (via internal.${DOMAIN} subdomains)
3. **Secure traffic routing** with automatic TLS
4. **Reliable networking** with proper load balancing

## Service Management

Wild Cloud uses a streamlined per-service setup approach:

**Primary Command**: `wild-service-setup <service> [options]`
- **Default**: Configure and deploy service using existing templates
- **`--fetch`**: Fetch fresh templates before setup (for updates)
- **`--no-deploy`**: Configure only, skip deployment (for planning)

**Master Orchestrator**: `wild-setup-services`
- Sets up all services in proper dependency order
- Each service validates its prerequisites before deployment
- Fail-fast approach with clear recovery instructions

## Architecture

```
Internet → External DNS → MetalLB LoadBalancer → Traefik → Kubernetes Services
                                    ↑
                                 Internal DNS
                                    ↑
                              Internal Network
```

## Key Components

- **[MetalLB](metallb/README.md)** - Provides load balancing for bare metal clusters
- **[Traefik](traefik/README.md)** - Handles ingress traffic, TLS termination, and routing
- **[cert-manager](cert-manager/README.md)** - Manages TLS certificates
- **[CoreDNS](coredns/README.md)** - Provides DNS resolution for services
- **[ExternalDNS](externaldns/README.md)** - Automatic DNS record management
- **[Longhorn](longhorn/README.md)** - Distributed storage system for persistent volumes
- **[NFS](nfs/README.md)** - Network file system for shared media storage (optional)
- **[Kubernetes Dashboard](kubernetes-dashboard/README.md)** - Web UI for cluster management (accessible via https://dashboard.internal.${DOMAIN})
- **[Docker Registry](docker-registry/README.md)** - Private container registry for custom images
- **[Utils](utils/README.md)** - Cluster utilities and debugging tools

## Common Usage Patterns

### Complete Infrastructure Setup
```bash
# All services with fresh templates (recommended for first-time setup)
wild-setup-services --fetch

# All services using existing templates (fastest)
wild-setup-services

# Configure all services but don't deploy (for planning)
wild-setup-services --no-deploy
```

### Individual Service Management
```bash
# Most common - reconfigure and deploy existing service
wild-service-setup cert-manager

# Get fresh templates and deploy (for updates)
wild-service-setup cert-manager --fetch

# Configure only, don't deploy (for planning)
wild-service-setup cert-manager --no-deploy

# Fresh templates + configure + deploy
wild-service-setup cert-manager --fetch
```

### Service Dependencies
Services are automatically deployed in dependency order:
1. **metallb** → Load balancing foundation
2. **traefik** → Ingress (requires metallb)
3. **cert-manager** → TLS certificates (requires traefik)
4. **externaldns** → DNS automation (requires cert-manager)
5. **kubernetes-dashboard** → Admin UI (requires cert-manager)

Each service validates its dependencies before deployment.

## Idempotent Design

All setup is designed to be idempotent and reliable:

- **Atomic Operations**: Each service handles its complete lifecycle
- **Dependency Validation**: Services check prerequisites before deployment
- **Error Recovery**: Failed services can be individually fixed and re-run
- **Safe Retries**: Operations can be repeated without harm
- **Incremental Updates**: Configuration changes applied cleanly

Example recovery from cert-manager failure:
```bash
# Fix the issue, then resume
wild-service-setup cert-manager --fetch
# Continue with remaining services
wild-service-setup externaldns --fetch
```
