# Wild Cloud App Lifecycle: State and Operations

## Overview

Wild Cloud manages applications across multiple independent systems with different consistency guarantees. Understanding these systems, their interactions, and how app packages are structured is critical for reliable app lifecycle management.

This document covers:
- **System architecture**: The three independent systems managing app state
- **User workflows**: Two distinct approaches (git-based vs Web UI)
- **App package structure**: How apps are defined in Wild Directory
- **State lifecycle**: Complete state transitions from add to delete
- **Operations**: How each lifecycle operation works across systems
- **Edge cases**: Common failure modes and automatic recovery

## User Workflows

Wild Cloud supports two fundamentally different workflows for managing app lifecycle:

### Advanced Users: Git-Based Infrastructure-as-Code

**Target Audience**: DevOps engineers, systems administrators, users comfortable with git and command-line tools.

**Key Characteristics**:
- Instance data directory is a git repository
- Wild Directory tracked as upstream remote
- Manual edits tracked in git with commit messages
- Wild Directory updates merged using standard git workflows
- Full version control and audit trail
- SSH/command-line access to Wild Central device

**Typical Workflow**:
```bash
# Clone instance repository
git clone user@wild-central:/var/lib/wild-central/instances/my-cloud

# Make custom changes
vim apps/myapp/deployment.yaml
git commit -m "Increase CPU limits for production"

# Merge upstream Wild Directory updates
git remote add wild-directory https://github.com/wildcloud/wild-directory.git
git fetch wild-directory
git merge wild-directory/main
# Resolve conflicts if needed

# Deploy changes
wild app deploy myapp
```

**Philosophy**: Treat cluster configuration like application code - version controlled, reviewed, tested, and deployed through established git workflows.

### Regular Users: Web UI-Based Management

**Target Audience**: Non-technical users, small teams, users who prefer graphical interfaces.

**Key Characteristics**:
- All management through Web UI or CLI (no SSH access)
- Configuration changes via forms (config.yaml, secrets.yaml)
- Wild Directory updates applied automatically with config merging
- Cannot directly edit manifest files (prevents divergence)
- Simplified workflow with automatic safety checks

**Typical Workflow**:
1. Browse available apps in Web UI
2. Click "Add" to add app to instance
3. Configure via form fields (port, storage, domain, etc.)
4. Click "Deploy" to deploy to cluster
5. System notifies when Wild Directory updates available
6. Click "Update" to merge changes (config preserved)
7. Review changes in diff view
8. Click "Deploy" to apply updates

**Philosophy**: Abstract away complexity - users manage apps like installing software, not like managing infrastructure code.

### Key Differences

| Aspect | Advanced Users (Git) | Regular Users (Web UI) |
|--------|----------------------|------------------------|
| **Access** | SSH + command line | Web UI + CLI |
| **Manifest Editing** | Direct file editing | Via config forms only |
| **Version Control** | Git (full history) | System managed |
| **Wild Directory Updates** | Manual git merge | Automatic merge with review |
| **Customization** | Unlimited | Configuration-based only |
| **Drift** | Intentional (git-tracked) | Unintentional (reconcile) |
| **Collaboration** | Git branches/PRs | Shared Web UI access |
| **Rollback** | `git revert` | Re-deploy previous state |

The rest of this document covers both workflows, with sections clearly marked for each user type where behavior differs.

## System Architecture

### The Multi-System Challenge

Wild Cloud app state spans **three independent systems**:

1. **Wild Directory** (Source of Truth)
   - Location: `/path/to/wild-directory/{app-name}/`
   - Consistency: Immutable, version controlled
   - Purpose: Template definitions shared across all instances

2. **Instance Data** (Local State)
   - Location: `/path/to/data-dir/instances/{instance}/`
   - Consistency: Immediately consistent, file-system based
   - Purpose: Instance-specific configuration and compiled manifests

3. **Kubernetes Cluster** (Runtime State)
   - Location: Kubernetes API and etcd
   - Consistency: Eventually consistent
   - Purpose: Running application workloads

**Critical Insight**: These systems have fundamentally different consistency models, creating inherent challenges for atomic operations across system boundaries.

## State Components

### 1. Wild Directory (Immutable Source)

```
wild-directory/
└── {app-name}/
    ├── manifest.yaml           # App metadata, dependencies, defaults
    ├── kustomization.yaml      # Kustomize configuration
    ├── deployment.yaml         # Kubernetes workload (template)
    ├── service.yaml           # Kubernetes service (template)
    ├── ingress.yaml           # Kubernetes ingress (template)
    ├── namespace.yaml         # Namespace definition (template)
    ├── pvc.yaml              # Storage claims (template)
    ├── db-init-job.yaml      # Database initialization (optional)
    └── README.md             # Documentation
```

**Characteristics**:
- Read-only during operations
- Contains gomplate template variables: `{{ .cloud.domain }}`, `{{ .app.port }}`
- Shared across all Wild Cloud instances
- Version controlled (git)

#### App Manifest Structure

The `manifest.yaml` file defines everything about an app:

```yaml
name: myapp                    # App identifier (matches directory name)
is: myapp                      # Unique app type identifier
description: Brief description
version: 1.0.0                # Follow upstream versioning
icon: https://example.com/icon.svg

requires:                     # Dependencies (optional)
  - name: postgres            # Dependency app type (matches 'is' field)
    alias: db                # Optional: reference name in templates
  - name: redis              # No alias = use 'redis' as reference

defaultConfig:                # Merged into instance config.yaml
  namespace: myapp
  image: myapp/myapp:latest
  port: "8080"
  storage: 10Gi
  domain: myapp.{{ .cloud.domain }}
  # Can reference dependencies:
  dbHost: "{{ .apps.db.host }}"
  redisHost: "{{ .apps.redis.host }}"

defaultSecrets:               # App's own secrets
  - key: apiKey              # Auto-generated random if no default
  - key: dbUrl               # Can use template with config/secrets
    default: "postgresql://{{ .app.dbUser }}:{{ .secrets.dbPassword }}@{{ .app.dbHost }}/{{ .app.dbName }}"

requiredSecrets:              # Secrets from dependencies
  - db.password              # Format: <app-ref>.<key>
  - redis.auth               # Copied from dependency's secrets
```

**Template Variable Resolution**:

In `manifest.yaml` only:
- `{{ .cloud.* }}` - Infrastructure config (domain, smtp, etc.)
- `{{ .cluster.* }}` - Cluster config (IPs, versions, etc.)
- `{{ .operator.* }}` - Operator info (email)
- `{{ .app.* }}` - This app's config from defaultConfig
- `{{ .apps.<ref>.* }}` - Dependency app's config (via requires mapping)
- `{{ .secrets.* }}` - This app's secrets (in defaultSecrets default only)

In `*.yaml` resource templates:
- `{{ .* }}` - Only this app's config (all from defaultConfig)
- No access to secrets, cluster config, or other apps

**Dependency Resolution**:
1. `requires` lists app types needed (matches `is` field)
2. At add time, user maps to actual installed apps
3. System stores mapping in `installedAs` field in instance manifest
4. Templates resolve `{{ .apps.db.* }}` using this mapping

### 2. Instance Data (Local State)

```
data-dir/instances/{instance}/
├── config.yaml               # App configuration (user-editable)
├── secrets.yaml             # App secrets (generated + user-editable)
├── kubeconfig               # Cluster access credentials
├── apps/
│   └── {app-name}/
│       ├── manifest.yaml    # Copy with installedAs mappings
│       ├── deployment.yaml  # Compiled (variables resolved)
│       ├── service.yaml     # Compiled
│       ├── ingress.yaml     # Compiled
│       └── ...             # All manifests compiled
└── operations/
    └── op_{action}_app_{app-name}_{timestamp}.json
```

#### config.yaml Structure

```yaml
apps:
  postgres:
    namespace: postgres
    image: pgvector/pgvector:pg15
    port: "5432"
    storage: 10Gi
    host: postgres.postgres.svc.cluster.local
    # ... all defaultConfig values from manifest
```

#### secrets.yaml Structure

```yaml
apps:
  postgres:
    password: <generated-random-32-chars>
  ghost:
    dbPassword: <generated>
    adminPassword: <generated>
    smtpPassword: <copied-from-smtp.password>
    # defaultSecrets + requiredSecrets
```

**Characteristics**:
- Immediately consistent (filesystem)
- File-locked during updates (`config.yaml.lock`, `secrets.yaml.lock`)
- Version controlled (recommended but optional)
- User-editable (advanced users can SSH and modify)

### 3. Kubernetes Cluster (Runtime State)

```
Kubernetes Cluster
└── Namespace: {app-name}
    ├── Deployment: {app-name}-*
    ├── ReplicaSet: {app-name}-*
    ├── Pod: {app-name}-*
    ├── Service: {app-name}
    ├── Ingress: {app-name}
    ├── PVC: {app-name}-pvc
    ├── Secret: {app-name}-secrets
    ├── ConfigMap: {app-name}-* (optional)
    └── Job: {app-name}-db-init (optional)
```

**Namespace Lifecycle**:
- `Active`: Normal operating state
- `Terminating`: Deletion in progress (may take time)
- Finalizers: `[kubernetes]` prevents deletion until resources cleaned

**Characteristics**:
- Eventually consistent (distributed system)
- Cascade deletion: Deleting namespace deletes all child resources
- Finalizers block deletion until cleared
- May enter stuck states requiring automatic intervention

#### Kubernetes Resource Labeling

All Wild Cloud apps use standard labels automatically applied via Kustomize:

```yaml
# In kustomization.yaml
labels:
  - includeSelectors: true    # Apply to resources AND selectors
    pairs:
      app: myapp              # App name
      managedBy: kustomize
      partOf: wild-cloud
```

This auto-expands selectors:
```yaml
# You write:
selector:
  component: web

# Kustomize expands to:
selector:
  app: myapp
  managedBy: kustomize
  partOf: wild-cloud
  component: web
```

**Important**: Use simple component labels (`component: web`), not Helm-style labels (`app.kubernetes.io/name`).

### 4. External System State (Kubernetes Controller-Managed)

These systems are not directly controlled by Wild Cloud API but are integral to app lifecycle:

#### External DNS (via external-dns controller)

**Location**: External DNS provider (Cloudflare, Route53, etc.)

**Trigger**: Ingress with external-dns annotations
```yaml
annotations:
  external-dns.alpha.kubernetes.io/target: {{ .domain }}
```

**State Flow**:
```
1. Deploy creates Ingress with annotations
2. external-dns controller watches Ingress resources
3. Controller creates DNS records at provider
4. DNS propagates (eventual consistency, 30-300 seconds)
5. Domain resolves to cluster load balancer IP
```

**Lifecycle**:
- **Create**: Automatic when Ingress deployed
- **Update**: Automatic when Ingress annotations change
- **Delete**: Automatic when Ingress deleted (DNS records cleaned up)

**Eventual Consistency**: DNS changes take 30s-5min to propagate globally.

**Edge Cases**:
- DNS propagation delay (app deployed but domain not resolving yet)
- Provider rate limits (too many updates)
- Stale records if external-dns controller down during deletion
- Multiple ingresses with same hostname (last write wins)

**Debugging**:
```bash
# View external-dns logs
kubectl logs -n external-dns deployment/external-dns

# Check what DNS records external-dns is managing
kubectl get ingress -A -o yaml | grep external-dns
```

#### TLS Certificates (via cert-manager)

**Location**: Both cluster (Kubernetes Secret) and external CA (Let's Encrypt)

**Trigger**: Ingress with cert-manager annotations
```yaml
annotations:
  cert-manager.io/cluster-issuer: letsencrypt-prod
spec:
  tls:
  - hosts:
    - myapp.cloud.example.com
    secretName: myapp-tls
```

**State Flow**:
```
1. Deploy creates Ingress with TLS config
2. cert-manager creates Certificate resource
3. cert-manager creates Order with ACME DNS-01 challenge
4. cert-manager updates DNS via provider (for challenge)
5. Let's Encrypt validates domain ownership via DNS
6. cert-manager receives certificate and stores in Secret
7. Ingress controller uses Secret for TLS termination
```

**Lifecycle**:
- **Create**: Automatic when Ingress with cert-manager annotation deployed
- **Renew**: Automatic (starts 30 days before expiry)
- **Delete**: Secret deleted with namespace, CA record persists

**Eventual Consistency**: Certificate issuance takes 30s-2min (DNS challenge + CA validation).

**Edge Cases**:
- DNS-01 challenge timeout (DNS not propagated yet)
- Rate limits (Let's Encrypt: 50 certs/domain/week, 5 failed validations/hour)
- Expired certificates (cert-manager should auto-renew but may fail)
- Namespace stuck terminating (cert-manager challenges may block finalizers)

**Debugging**:
```bash
# View certificates and their status
kubectl get certificate -n myapp
kubectl describe certificate myapp-tls -n myapp

# View ACME challenge progress
kubectl get certificaterequest -n myapp
kubectl get order -n myapp
kubectl get challenge -n myapp

# Check cert-manager logs
kubectl logs -n cert-manager deployment/cert-manager
```

#### Wildcard Certificates (Shared Resource Pattern)

Wild Cloud uses **two shared wildcard certificates** to avoid rate limits:

**1. Public Wildcard Certificate** (`wildcard-wild-cloud-tls`)
```yaml
# Created once in cert-manager namespace
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: wildcard-wild-cloud-tls
  namespace: cert-manager
spec:
  secretName: wildcard-wild-cloud-tls
  dnsNames:
  - "*.cloud.example.com"
```

**2. Internal Wildcard Certificate** (`wildcard-internal-wild-cloud-tls`)
```yaml
# For internal-only apps not exposed via external-dns
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: wildcard-internal-wild-cloud-tls
  namespace: cert-manager
spec:
  secretName: wildcard-internal-wild-cloud-tls
  dnsNames:
  - "*.internal.cloud.example.com"
```

**Usage Pattern**:
- **Public apps** (exposed externally): Use `wildcard-wild-cloud-tls`
  - Domain: `myapp.cloud.example.com`
  - Has external-dns annotation (creates public DNS record)

- **Internal apps** (cluster-only): Use `wildcard-internal-wild-cloud-tls`
  - Domain: `myapp.internal.cloud.example.com`
  - No external-dns annotation (only accessible within cluster/LAN)
  - Examples: Docker registry, internal dashboards

**Shared Pattern**:
1. One wildcard cert per domain covers all subdomains
2. Apps reference via `tlsSecretName: wildcard-wild-cloud-tls` (or `wildcard-internal-wild-cloud-tls`)
3. Deploy operation copies secret from cert-manager namespace to app namespace
4. All apps on same domain share the certificate

**Advantages**:
- Avoids Let's Encrypt rate limits (50 certs/domain/week)
- Faster deployment (no ACME challenge per app)
- Survives app delete/redeploy (cert persists in cert-manager namespace)

**Trade-offs**:
- All apps on same domain share same cert (if compromised, affects all apps)
- Cert must be copied to each app namespace (handled by Deploy operation)

**Copy Operation**:
```go
// In apps.Deploy()
// Copies both wildcard certs if referenced by ingress
wildcardSecrets := []string{"wildcard-wild-cloud-tls", "wildcard-internal-wild-cloud-tls"}
for _, secretName := range wildcardSecrets {
    if bytes.Contains(ingressContent, []byte(secretName)) {
        utilities.CopySecretBetweenNamespaces(kubeconfigPath, secretName, "cert-manager", appName)
    }
}
```

#### Load Balancer IPs (via MetalLB)

**Location**: MetalLB controller state + cluster network

**Trigger**: Service with `type: LoadBalancer`
```yaml
apiVersion: v1
kind: Service
metadata:
  name: traefik
  namespace: traefik
spec:
  type: LoadBalancer
  loadBalancerIP: 192.168.8.80  # Optional: request specific IP
```

**State Flow**:
```
1. Service created with type: LoadBalancer
2. MetalLB controller assigns IP from configured pool
3. MetalLB announces IP via ARP (Layer 2) or BGP (Layer 3)
4. Network routes traffic to assigned IP
5. kube-proxy on nodes routes to service endpoints
```

**Lifecycle**:
- **Create**: Automatic when LoadBalancer Service deployed
- **Persist**: IP sticky (same IP across pod restarts)
- **Delete**: IP returned to pool when Service deleted

**Eventual Consistency**: ARP cache clearing takes 0-60 seconds.

**Edge Cases**:
- IP pool exhaustion (no IPs available from MetalLB pool)
- IP conflicts (pool overlaps with DHCP or static assignments)
- ARP cache issues (old MAC address cached, traffic fails until cleared)
- Split-brain scenarios (multiple nodes announce same IP)

**Debugging**:
```bash
# View services with assigned IPs
kubectl get svc -A --field-selector spec.type=LoadBalancer

# Check MetalLB IP pools
kubectl get ipaddresspool -n metallb-system

# View MetalLB controller state
kubectl logs -n metallb-system deployment/controller
kubectl logs -n metallb-system daemonset/speaker
```

### Cross-System Dependency Chain

A complete app deployment triggers this cascade across systems:

```
Wild Cloud API (Deploy)
  ↓
Kubernetes (kubectl apply)
  ↓
Namespace + Resources Created
  ↓
┌─────────────────┬──────────────────┬─────────────────┐
│                 │                  │                 │
external-dns    cert-manager      MetalLB
watches Ingress  watches Ingress   watches Service
  ↓                 ↓                  ↓
DNS Provider      Let's Encrypt     Network ARP/BGP
(Cloudflare)      (ACME CA)         (Local Network)
  ↓                 ↓                  ↓
CNAME Record      TLS Certificate   IP Address
Created           Issued            Announced
(30s-5min)        (30s-2min)        (0-60s)
  ↓                 ↓                  ↓
Domain Resolves + HTTPS Works + Traffic Routes
```

**Total Time to Fully Operational**:
- Kubernetes resources: 5-30 seconds (image pull + pod start)
- DNS propagation: 30 seconds - 5 minutes
- TLS certificate: 30 seconds - 2 minutes
- Network ARP: 0-60 seconds

**Worst case**: 5-7 minutes from deploy command to app fully accessible via HTTPS.

## App Lifecycle States

### State 0: NOT_ADDED

```
Wild Directory:     {app-name}/ exists (templates)
Instance Apps:      (does not exist)
config.yaml:        (no apps.{app-name} entry)
secrets.yaml:       (no apps.{app-name} entry)
Cluster:            (no namespace)
```

**Invariants**:
- App can be added from Wild Directory
- No local or cluster state exists

---

### State 1: ADDED

**After**: `wild app add {app-name}`

```
Wild Directory:     {app-name}/ (unchanged)
Instance Apps:      {app-name}/ created with compiled manifests
                    manifest.yaml has installedAs dependency mappings
config.yaml:        apps.{app-name} populated from defaultConfig
secrets.yaml:       apps.{app-name} populated with generated secrets
Cluster:            (no namespace yet)
```

**Operations**:
1. Read `wild-directory/{app-name}/manifest.yaml`
2. Resolve gomplate variables using instance config
3. Generate random secrets for `defaultSecrets` (if no default provided)
4. Copy secrets from dependencies for `requiredSecrets`
5. Compile templates → write to `instance/apps/{app-name}/`
6. Append to `config.yaml` (file-locked)
7. Append to `secrets.yaml` (file-locked)

**Invariants**:
- Local state consistent: config, secrets, and compiled manifests all exist
- Cluster state empty: nothing deployed yet
- Idempotent: Can re-add without side effects (overwrites local state)

---

### State 2: DEPLOYING

**During**: `wild app deploy {app-name}`

```
Wild Directory:     (unchanged)
Instance Apps:      (unchanged)
config.yaml:        (unchanged)
secrets.yaml:       (unchanged)
Cluster:            namespace: Active (or being created)
                    resources: Creating/Pending
                    secret/{app-name}-secrets: Created
```

**Operations**:
1. Check namespace status (pre-flight check)
2. Create/update namespace (idempotent)
3. Create Kubernetes secret from `secrets.yaml` (overwrite if exists)
4. Copy dependency secrets (e.g., postgres-secrets)
5. Copy TLS certificates (e.g., wildcard certs from cert-manager)
6. Apply manifests: `kubectl apply -k instance/apps/{app-name}/`

**Invariants Being Established**:
- Namespace must be `Active` or `NotFound` (not `Terminating`)
- Kubernetes secret created before workloads
- All dependencies deployed first

---

### State 3: DEPLOYED

**After successful deploy**:

```
Wild Directory:     (unchanged)
Instance Apps:      (unchanged)
config.yaml:        (unchanged)
secrets.yaml:       (unchanged)
Cluster:            namespace: Active
                    deployment: Ready (replicas running)
                    pods: Running
                    service: Available (endpoints ready)
                    ingress: Ready (external-dns created DNS)
                    pvc: Bound (storage provisioned)
                    secret: Exists
```

**Invariants**:
- **Strong consistency**: Local state matches cluster intent
- All pods healthy and running
- Services have endpoints
- DNS records created (via external-dns)
- TLS certificates valid (via cert-manager)

**Health Checks**:
```bash
kubectl get pods -n {app-name}
kubectl get ingress -n {app-name}
kubectl get pvc -n {app-name}
```

---

### State 3a: UPDATING (Configuration/Secret Changes)

**Scenario**: User modifies config.yaml or secrets.yaml and redeploys.

**Operations**:

#### Update Configuration Only
```
1. User edits config.yaml (e.g., change port, storage size)
2. User runs: wild app deploy {app-name}
3. System re-compiles templates with new config
4. System applies updated manifests: kubectl apply -k
5. Kubernetes performs rolling update (if applicable)
```

**State Flow**:
```
config.yaml:        Modified (new values)
Instance Apps:      Templates re-compiled with new config
secrets.yaml:       (unchanged)
Cluster:            Rolling update (pods recreated with new config)
```

**Important**: Config changes trigger template recompilation. The `.package` directory preserves original templates, but deployed manifests are regenerated.

#### Update Secrets Only
```
1. User edits secrets.yaml (e.g., change password)
2. User runs: wild app deploy {app-name}
3. System deletes old Kubernetes secret
4. System creates new Kubernetes secret with updated values
5. Pods must be restarted to pick up new secrets
```

**State Flow**:
```
config.yaml:        (unchanged)
Instance Apps:      (unchanged - no template changes)
secrets.yaml:       Modified (new secrets)
Cluster:            Secret updated, pods may need manual restart
```

**Critical**: Most apps don't auto-reload secrets. May require manual pod restart:
```bash
kubectl rollout restart deployment/{app-name} -n {app-name}
```

#### Update Both Config and Secrets
```
1. User edits both config.yaml and secrets.yaml
2. User runs: wild app deploy {app-name}
3. System re-compiles templates + updates secrets
4. System applies manifests (rolling update)
5. Pods restart with new config and secrets
```

---

### State 3b: UPDATING (Manifest/Template Changes)

**Scenario**: User directly edits Kustomize files in instance apps directory.

This workflow differs significantly for **advanced users** (git-based) vs **regular users** (Web UI/CLI).

#### Advanced User Workflow (Git-Based)

**Instance directory as git repository**:
```bash
# Instance data directory is a git repo
cd /var/lib/wild-central/instances/my-cloud
git status
git log
```

**Operations**:
```
1. User SSHs to Wild Central device (or uses VSCode Remote SSH)
2. User edits: apps/{app-name}/deployment.yaml
3. User commits changes: git add . && git commit -m "Custom resource limits"
4. User runs: wild app deploy {app-name} OR kubectl apply -k apps/{app-name}/
5. Changes applied to cluster
```

**State Flow**:
```
Wild Directory:     (unchanged - original templates intact)
Instance Apps:      Modified and git-tracked (intentional divergence)
config.yaml:        (unchanged)
secrets.yaml:       (unchanged)
Cluster:            Updated with manual changes
.package/:          (unchanged - preserves original templates)
Git History:        Tracks all manual edits with commit messages
```

**Benefits**:
- **Version Control**: Full audit trail of all changes
- **Rollback**: `git revert` to undo changes
- **Infrastructure as Code**: Instance config managed like application code
- **Collaboration**: Multiple admins can work on same cluster config
- **Merge Workflow**: Wild Directory updates handled as upstream merges

**Example Git Workflow**:
```bash
# Make custom changes
vim apps/myapp/deployment.yaml
git add apps/myapp/deployment.yaml
git commit -m "Increase CPU limit for production load"

# Deploy changes
wild app deploy myapp

# Later, merge upstream Wild Directory updates
git pull upstream main  # Pull Wild Directory changes
git merge upstream/main  # Merge with local customizations
# Resolve any conflicts
git push origin main
```

#### Regular User Workflow (Web UI/CLI)

**Operations**:
```
1. User cannot directly edit manifests (no SSH access)
2. User modifies config.yaml or secrets.yaml via Web UI
3. System re-compiles templates automatically
4. User deploys via Web UI
```

**State Flow**:
```
Wild Directory:     (unchanged)
Instance Apps:      Re-compiled from templates (stays in sync)
config.yaml:        Modified via Web UI
secrets.yaml:       Modified via Web UI
Cluster:            Updated via Web UI deploy
```

**Protection**:
- No manual manifest editing (prevents divergence)
- All changes through config/secrets (stays synchronized)
- Wild Directory updates apply cleanly (no merge conflicts)

---

### State 3c: UPDATING (Wild Directory Version Update)

**Scenario**: Wild Directory app updated (bug fix, new version, new features).

This workflow differs significantly for **advanced users** (git-based) vs **regular users** (Web UI/CLI).

#### Advanced User Workflow (Git Merge)

**Wild Directory as upstream remote**:
```bash
# Add Wild Directory as upstream remote (one-time setup)
git remote add wild-directory https://github.com/wildcloud/wild-directory.git
git fetch wild-directory
```

**Detection**:
```bash
# Check for upstream updates
git fetch wild-directory
git log HEAD..wild-directory/main --oneline

# See what changed in specific app
git diff HEAD wild-directory/main -- apps/myapp/
```

**Merge Operations**:
```bash
# 1. Fetch latest Wild Directory changes
git fetch wild-directory

# 2. Merge upstream changes with local customizations
git merge wild-directory/main

# 3. Resolve any conflicts
# Git will show conflicts in manifest files, config, etc.
# User resolves conflicts preserving their custom changes

# 4. Test changes
wild app deploy myapp --dry-run

# 5. Deploy updated app
wild app deploy myapp

# 6. Commit merge
git push origin main
```

**Conflict Resolution Example**:
```yaml
# Conflict in apps/myapp/deployment.yaml
<<<<<<< HEAD
      resources:
        limits:
          cpu: "2000m"      # Local customization
          memory: "4Gi"     # Local customization
=======
      resources:
        limits:
          cpu: "1000m"      # Wild Directory default
          memory: "2Gi"     # Wild Directory default
>>>>>>> wild-directory/main
```

User resolves by keeping their custom values or adopting new defaults.

**Benefits**:
- **Full Control**: User decides what to merge and when
- **Conflict Resolution**: Git's standard merge tools handle conflicts
- **Audit Trail**: Git history shows what changed and why
- **Selective Updates**: Can cherry-pick specific app updates

**State Flow**:
```
Wild Directory:     (tracked as remote, fetched regularly)
Instance Apps:      Merged with git (custom + upstream changes)
config.yaml:        Manually merged (conflicts resolved by user)
secrets.yaml:       Preserved (not in Wild Directory)
.package/:          Updated after merge
Git History:        Shows merge commits and conflict resolutions
Cluster:            Updated when user deploys after merge
```

#### Regular User Workflow (Automated Merge)

**Detection Methods**:

**Method 1: Compare .package with Wild Directory**
```bash
# System compares checksums/timestamps
diff -r instance/apps/{app-name}/.package/ wild-directory/{app-name}/
```

If differences exist: New version available in Wild Directory.

**Method 2: Check manifest version field**
```yaml
# wild-directory/{app-name}/manifest.yaml
version: 2.0.0

# instance/apps/{app-name}/manifest.yaml
version: 1.0.0  # Older version
```

**Safe Update (Preserves Local Config)**
```
1. System detects Wild Directory changes
2. User initiates update (via UI or CLI)
3. System backs up current instance state:
   - Saves current config.yaml section
   - Saves current secrets.yaml section
   - Saves current manifest.yaml (with installedAs mappings)
4. System re-adds app from Wild Directory:
   - Copies new templates to instance/apps/{app-name}/
   - Updates .package/ with new source files
   - Merges new defaultConfig with existing config
   - Preserves existing secrets (doesn't regenerate)
5. System re-compiles templates with preserved config
6. User reviews changes (diff shown in UI)
7. User deploys updated app
```

**State Flow**:
```
Wild Directory:     (unchanged - new version available)
Instance Apps:      Updated templates + recompiled manifests
config.yaml:        Merged (new fields added, existing preserved)
secrets.yaml:       (unchanged - existing secrets preserved)
.package/:          Updated with new source files
Cluster:            (not changed until user deploys)
```

**Merge Strategy for Config**:
```yaml
# Old config.yaml (version 1.0.0)
apps:
  myapp:
    port: "8080"
    storage: 10Gi

# New Wild Directory manifest (version 2.0.0) adds "replicas" field
defaultConfig:
  port: "8080"
  storage: 10Gi
  replicas: "3"  # New field

# Merged config.yaml (after update)
apps:
  myapp:
    port: "8080"      # Preserved
    storage: 10Gi     # Preserved
    replicas: "3"     # Added
```

**Breaking Changes**:
If Wild Directory update has breaking changes (renamed fields, removed features):
- System cannot auto-merge
- User must manually reconcile
- UI shows conflicts and requires resolution

#### Destructive Update (Fresh Install)
```
1. User deletes app: wild app delete {app-name}
2. User re-adds app: wild app add {app-name}
3. Config and secrets regenerated (loses customizations)
4. User must manually reconfigure
```

**Use When**:
- Major version upgrade with breaking changes
- Significant manifest restructuring
- User wants clean slate

---

### State 3d: DEPLOYED with Drift

**Scenario**: Cluster state diverged from instance state.

This state has different meanings for **advanced users** vs **regular users**.

#### Advanced Users: Intentional Drift (Git-Tracked)

**Scenario**: User made direct cluster changes and committed them to git.

**Example**:
```bash
# User edits deployment directly
kubectl edit deployment myapp -n myapp

# User documents change in git
vim apps/myapp/deployment.yaml  # Update manifest to match
git add apps/myapp/deployment.yaml
git commit -m "Emergency CPU limit increase for production incident"
```

**State Flow**:
```
Instance Apps:      Updated and git-tracked (intentional)
Git History:        Documents why change was made
Cluster:            Matches updated instance state
```

**This is NOT drift** - it's infrastructure-as-code in action. The instance directory reflects the true desired state, tracked in git.

**Reconciliation**: Not needed (intentional state).

#### Regular Users: Unintentional Drift

**Scenario**: Cluster state diverged from instance state (unexpected).

**Causes**:
- User ran `kubectl edit` directly (shouldn't happen - no SSH access)
- Another admin modified cluster resources
- Partial deployment failure (some resources applied, others failed)
- Kubernetes controller modified resources (e.g., HPA changed replicas)

**Detection**:
```bash
# Compare desired vs actual state
kubectl diff -k instance/apps/{app-name}/

# Or use declarative check
kubectl apply -k instance/apps/{app-name}/ --dry-run=server
```

**State Flow**:
```
Instance Apps:      Unchanged (desired state)
Cluster:            Diverged (actual state differs)
```

**Reconciliation**:
```
1. User runs: wild app deploy {app-name}
2. kubectl apply re-applies desired state
3. Kubernetes reconciles differences (three-way merge)
4. Cluster returns to matching instance state
```

**Important**: `kubectl apply` is idempotent and safe for reconciliation.

#### Distinguishing Intentional vs Unintentional Drift

**Advanced users (git-based)**:
- Check git status: `git status` shows no uncommitted changes → intentional
- Check git log: `git log -- apps/myapp/` shows recent commits → intentional
- Cluster state matches git-tracked files → intentional

**Regular users (Web UI)**:
- Any divergence is unintentional (no way to edit manifests directly)
- Reconcile immediately by redeploying

---

### State 4: DELETING

**During**: `wild app delete {app-name}`

```
Wild Directory:     (unchanged)
Instance Apps:      Being removed
config.yaml:        apps.{app-name} being removed
secrets.yaml:       apps.{app-name} being removed
Cluster:            namespace: Active → Terminating
                    resources: Deleting (cascade)
```

**Operations** (Two-Phase):

**Phase 1: Cluster Cleanup (Best Effort)**
```bash
# Try graceful deletion
kubectl delete namespace {app-name} --timeout=30s --wait=true

# If stuck, force cleanup
kubectl patch namespace {app-name} --type=merge -p '{"metadata":{"finalizers":null}}'
```

**Phase 2: Local Cleanup (Always Succeeds)**
```bash
rm -rf instance/apps/{app-name}/
yq delete config.yaml '.apps.{app-name}'
yq delete secrets.yaml '.apps.{app-name}'
```

**Critical Design Decision**:
- **Don't wait indefinitely for cluster cleanup**
- Local state is immediately consistent after Phase 2
- Cluster cleanup is eventually consistent

---

### State 5: DELETED

**After successful delete**:

```
Wild Directory:     (unchanged - still available for re-add)
Instance Apps:      (removed)
config.yaml:        (no apps.{app-name} entry)
secrets.yaml:       (no apps.{app-name} entry)
Cluster:            namespace: NotFound
                    all resources: (removed)
```

**Invariants**:
- Local state has no trace of app
- Cluster has no namespace or resources
- App can be re-added cleanly

---

### State X: STUCK_TERMINATING (Edge Case)

**Problematic state when namespace won't delete**:

```
Wild Directory:     (unchanged)
Instance Apps:      May or may not exist (depends on delete progress)
config.yaml:        May or may not have entry
secrets.yaml:       May or may not have entry
Cluster:            namespace: Terminating (STUCK!)
                    finalizers: Blocking deletion
                    resources: Some exist, some terminating
```

**Why This Happens**:
1. Resources with custom finalizers
2. Webhooks or admission controllers blocking deletion
3. Network issues during deletion
4. StatefulSet with orphaned PVCs

**Resolution**:
- Handled automatically by Deploy pre-flight checks
- Force cleanup finalizers after retries
- User never needs manual intervention

## System Boundaries and Consistency

### Consistency Guarantees by System

| System | Consistency Model | Synchronization |
|--------|------------------|-----------------|
| Wild Directory | Immutable | Read-only |
| Instance Data | Immediately Consistent | File locks |
| Kubernetes | Eventually Consistent | Reconciliation loops |

### Cross-System Operations

#### Delete Operation (Spans 2 Systems)

```
┌─────────────────────────────────────────────────┐
│ Delete Operation Timeline                       │
├─────────────────────────────────────────────────┤
│                                                 │
│ T=0s:  kubectl delete namespace (initiated)    │
│        └─ Cluster enters eventual consistency   │
│                                                 │
│ T=1s:  rm apps/{app-name}/ (completes)        │
│        yq delete config.yaml (completes)        │
│        yq delete secrets.yaml (completes)       │
│        └─ Local state immediately consistent    │
│                                                 │
│ T=2s:  Return success to user                  │
│                                                 │
│ T=30s: Namespace still terminating in cluster  │
│        └─ This is OK! Eventually consistent     │
│                                                 │
│ T=60s: Cluster cleanup completes               │
│        └─ Both systems now consistent           │
└─────────────────────────────────────────────────┘
```

**Key Insight**: We accept temporary inconsistency at the system boundary.

#### Deploy Operation (Spans 2 Systems)

```
┌─────────────────────────────────────────────────┐
│ Deploy Operation Timeline                       │
├─────────────────────────────────────────────────┤
│                                                 │
│ T=0s:  Check namespace status (pre-flight)     │
│        If Terminating: Force cleanup + retry    │
│                                                 │
│ T=5s:  Create namespace (idempotent)           │
│        Create secrets                           │
│        Apply manifests                          │
│        └─ Cluster enters reconciliation         │
│                                                 │
│ T=30s: Pods starting, images pulling           │
│                                                 │
│ T=60s: All pods Running, services ready        │
│        └─ Deployment successful                 │
└─────────────────────────────────────────────────┘
```

**Key Insight**: Deploy owns making cluster match local state.

## Idempotency and Safety

### Idempotent Operations

| Operation | Idempotent? | Why |
|-----------|-------------|-----|
| `app add` | ✅ Yes | Overwrites local state |
| `app deploy` | ✅ Yes | `kubectl apply` is idempotent |
| `app delete` | ✅ Yes | `kubectl delete --ignore-not-found` |

### Non-Idempotent Danger Zones

1. **Secret Generation**: Regenerating secrets breaks running apps
   - Solution: Only generate if key doesn't exist

2. **Database Initialization**: Running twice can cause conflicts
   - Solution: Job uses `CREATE IF NOT EXISTS`, `ALTER IF EXISTS`

3. **Finalizer Removal**: Skips cleanup logic
   - Solution: Only as last resort after graceful attempts

## Edge Cases and Error Handling

### Edge Case 1: Namespace Stuck Terminating

**Scenario**: Previous delete left namespace in Terminating state.

**Detection**:
```bash
kubectl get namespace {app-name} -o jsonpath='{.status.phase}'
# Returns: "Terminating"
```

**Resolution** (Automatic):
1. Deploy pre-flight check detects Terminating state
2. Attempts force cleanup: removes finalizers
3. Waits 5 seconds
4. Retries up to 3 times
5. If still stuck, returns clear error message

**Code**:
```go
if status == "Terminating" {
    forceNamespaceCleanup(kubeconfigPath, appName)
    time.Sleep(5 * time.Second)
    // Retry deploy
}
```

### Edge Case 2: Concurrent Delete + Deploy

**Scenario**: User deletes app, then immediately redeploys.

**Timeline**:
```
T=0s:  Delete initiated
T=1s:  Local state cleaned up
T=2s:  User clicks "Deploy"
T=3s:  Deploy detects Terminating namespace
T=4s:  Deploy force cleanups and retries
T=10s: Deploy succeeds
```

**Why This Works**:
- Delete doesn't block on cluster cleanup
- Deploy handles any namespace state
- Eventual consistency at system boundary

### Edge Case 3: Dependency Not Deployed

**Scenario**: User tries to deploy app requiring postgres, but postgres isn't deployed.

**Current Behavior**: Deployment succeeds but pods crash (CrashLoopBackOff).

**Detection**:
```bash
kubectl get pods -n {app-name}
# Shows: CrashLoopBackOff
kubectl logs {pod-name} -n {app-name}
# Shows: "Connection refused to postgres.postgres.svc.cluster.local"
```

**Future Enhancement**: Pre-flight dependency check in Deploy operation.

### Edge Case 4: Secrets Out of Sync

**Scenario**: User manually updates password in Kubernetes but not in `secrets.yaml`.

**Impact**:
- Next deploy overwrites Kubernetes secret
- App may lose access if password changed elsewhere

**Best Practice**: Always update `secrets.yaml` as source of truth.

### Edge Case 5: PVC Retention

**Scenario**: Delete removes namespace but PVCs may persist (depends on reclaim policy).

**Behavior**:
- PVC with `ReclaimPolicy: Retain` stays after delete
- Redeploy creates new PVC (data orphaned)

**Resolution**: Document PVC backup/restore procedures.

## App Package Development Best Practices

### Security Requirements

**All pods must include security contexts**:

```yaml
spec:
  template:
    spec:
      securityContext:
        runAsNonRoot: true
        runAsUser: 999         # Use appropriate non-root UID
        runAsGroup: 999
        seccompProfile:
          type: RuntimeDefault
      containers:
      - name: app
        securityContext:
          allowPrivilegeEscalation: false
          capabilities:
            drop: [ALL]
          readOnlyRootFilesystem: false  # true when possible
```

Common UIDs: PostgreSQL/Redis use 999.

### Database Initialization Pattern

Apps requiring databases should include `db-init-job.yaml`:

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: myapp-db-init
spec:
  template:
    spec:
      restartPolicy: OnFailure
      containers:
      - name: db-init
        image: postgres:15
        command:
        - /bin/bash
        - -c
        - |
          # Create database if doesn't exist
          # Create/update user with password
          # Grant permissions
```

**Critical**: Use idempotent SQL:
- `CREATE DATABASE IF NOT EXISTS`
- `CREATE USER IF NOT EXISTS ... ELSE ALTER USER ... WITH PASSWORD`
- Jobs retry on failure until success

### Database URL Secrets

Never use runtime variable substitution - it doesn't work with Kustomize:

```yaml
# ❌ Wrong:
- name: DB_URL
  value: "postgres://user:$(DB_PASSWORD)@host/db"

# ✅ Correct:
- name: DB_URL
  valueFrom:
    secretKeyRef:
      name: myapp-secrets
      key: dbUrl
```

Define `dbUrl` in manifest's `defaultSecrets` with template:
```yaml
defaultSecrets:
  - key: dbUrl
    default: "postgres://{{ .app.dbUser }}:{{ .secrets.dbPassword }}@{{ .app.dbHost }}/{{ .app.dbName }}"
```

### External DNS Integration

Ingresses should include external-dns annotations:

```yaml
metadata:
  annotations:
    external-dns.alpha.kubernetes.io/target: {{ .domain }}
    external-dns.alpha.kubernetes.io/cloudflare-proxied: "false"
```

This creates: `myapp.cloud.example.com` → `cloud.example.com` (CNAME)

### Converting from Helm Charts

1. Extract and render Helm chart:
   ```bash
   helm fetch --untar --untardir charts repo/chart-name
   helm template --output-dir base --namespace myapp myapp charts/chart-name
   ```

2. Create Wild Cloud structure:
   - Add `namespace.yaml`
   - Run `kustomize create --autodetect`
   - Create `manifest.yaml`
   - Replace values with gomplate variables
   - Update labels (remove Helm-style, add Wild Cloud standard)
   - Add security contexts
   - Add external-dns annotations

## Testing Strategies

### Unit Tests

Test individual operations in isolation:

```go
func TestDelete_NamespaceNotFound(t *testing.T) {
    // Test delete when namespace doesn't exist
    // Should succeed without error
}

func TestDelete_NamespaceTerminating(t *testing.T) {
    // Test delete when namespace stuck terminating
    // Should force cleanup and succeed
}

func TestDeploy_NamespaceTerminating(t *testing.T) {
    // Test deploy when namespace terminating
    // Should retry and eventually succeed
}
```

### Integration Tests

Test cross-system operations:

```go
func TestDeleteThenDeploy(t *testing.T) {
    // 1. Deploy app
    // 2. Delete app
    // 3. Immediately redeploy
    // Should succeed without manual intervention
}

func TestConcurrentOperations(t *testing.T) {
    // Test multiple operations on same app
    // File locks should prevent corruption
}
```

### Chaos Tests

Test resilience to failures:

```go
func TestDeleteWithNetworkPartition(t *testing.T) {
    // Simulate network failure during delete
    // Local state should still be cleaned up
}

func TestDeployWithStuckFinalizer(t *testing.T) {
    // Manually add finalizer to namespace
    // Deploy should detect and force cleanup
}
```

## Operational Procedures

### Manual Inspection

**Check all state locations**:
```bash
# 1. Local state
ls instance/apps/{app-name}/
yq eval '.apps.{app-name}' config.yaml
yq eval '.apps.{app-name}' secrets.yaml

# 2. Cluster state
kubectl get namespace {app-name}
kubectl get all -n {app-name}
kubectl get pvc -n {app-name}
kubectl get secrets -n {app-name}
kubectl get ingress -n {app-name}
```

**Check operation status**:
```bash
ls -lt instance/operations/ | head -5
cat instance/operations/op_deploy_app_{app-name}_*.json
```

### Manual Recovery

**If namespace stuck terminating**:
```bash
# This should never be needed - Deploy handles automatically
# But for understanding:
kubectl get namespace {app-name} -o json | \
  jq '.spec.finalizers = []' | \
  kubectl replace --raw /api/v1/namespaces/{app-name}/finalize -f -
```

**If local state corrupted**:
```bash
# Re-add from Wild Directory
wild app add {app-name}
# This regenerates local state from source
```

**If secrets lost**:
```bash
# Secrets are auto-generated on add
# If lost, must re-add app (regenerates new secrets)
# Apps will need reconfiguration with new credentials
```

## Design Principles

### 1. Eventual Consistency at Boundaries

Accept that cluster state and local state may temporarily diverge. Design operations to handle any state.

### 2. Local State as Source of Truth

Instance data (config.yaml, secrets.yaml) is authoritative for intended state. Cluster reflects current state.

### 3. Idempotent Everything

Every operation should be safely repeatable. Use:
- `kubectl apply` (not `create`)
- `kubectl delete --ignore-not-found`
- `CREATE IF NOT EXISTS` in SQL

### 4. Fail Forward, Not Backward

If operation partially completes, retry should make progress (not start over).

### 5. No Indefinite Waits

Operations timeout and fail explicitly rather than hanging forever.

### 6. User Never Needs Manual Intervention

Automated recovery from all known edge cases (stuck namespaces, etc.).

## Future Enhancements

### 1. Dependency Validation

Pre-flight check that required apps are deployed:
```go
if manifest.Requires != nil {
    for _, dep := range manifest.Requires {
        if !isAppDeployed(dep.Name) {
            return fmt.Errorf("dependency %s not deployed", dep.Name)
        }
    }
}
```

### 2. State Reconciliation

Periodic background job to ensure consistency:
```go
func ReconcileAppState(appName string) {
    localState := readLocalState(appName)
    clusterState := readClusterState(appName)

    if !statesMatch(localState, clusterState) {
        // Alert or auto-correct
    }
}
```

### 3. Backup/Restore Workflows

Built-in PVC backup before delete:
```bash
wild app backup {app-name}
wild app restore {app-name} --from-backup {timestamp}
```

### 4. Dry-Run Mode

Preview changes without applying:
```bash
wild app deploy {app-name} --dry-run
# Shows: resources that would be created/updated
```

## Git Workflow Best Practices (Advanced Users)

This section provides operational guidance for advanced users managing Wild Cloud instances as git repositories.

### Initial Repository Setup

```bash
# Initialize instance directory as git repo
cd /var/lib/wild-central/instances/my-cloud
git init
git add .
git commit -m "Initial Wild Cloud instance configuration"

# Add Wild Directory as upstream remote
git remote add wild-directory https://github.com/wildcloud/wild-directory.git
git fetch wild-directory

# Add origin for your team's instance repo
git remote add origin git@github.com:myorg/wild-cloud-instances.git
git push -u origin main
```

### .gitignore Configuration

```bash
# Create .gitignore for instance directory
cat > .gitignore <<EOF
# Never commit secrets
secrets.yaml

# Kubernetes runtime state (cluster manages this)
.kube/cache/

# Temporary files
*.tmp
*.swp
.DS_Store

# Logs
*.log
EOF
```

**Critical**: `secrets.yaml` must NEVER be committed to git. Manage secrets through secure channels (password manager, Vault, etc.).

### Branch Strategy

**Recommended: Gitflow-style branching**

```bash
# Main branch = production state
main

# Development branch = testing changes
develop

# Feature branches = specific changes
feature/increase-postgres-storage
feature/add-monitoring-app
hotfix/fix-ghost-domain
```

**Workflow**:
```bash
# Make changes on feature branch
git checkout -b feature/add-redis
wild app add redis
git add apps/redis/ config.yaml
git commit -m "Add Redis cache for production load"

# Test deployment
wild app deploy redis

# Merge to develop for staging
git checkout develop
git merge feature/add-redis
git push origin develop

# Deploy to staging cluster
wild app deploy redis

# After testing, merge to main
git checkout main
git merge develop
git push origin main
```

### Merging Wild Directory Updates

**Regular update workflow**:

```bash
# Check for upstream updates (weekly/monthly)
git fetch wild-directory
git log HEAD..wild-directory/main --oneline

# Review what changed
git diff HEAD wild-directory/main -- apps/

# Merge updates
git checkout -b wild-directory-update-2024-01
git merge wild-directory/main

# Resolve conflicts (if any)
# Conflicts typically in:
# - apps/{app}/manifest.yaml (version numbers)
# - apps/{app}/deployment.yaml (resource limits, image versions)
# - config.yaml (new default fields)

# Test changes
wild app deploy {updated-apps} --dry-run
wild app deploy {updated-apps}

# Push updates
git push origin wild-directory-update-2024-01

# Create PR for team review
```

**Conflict resolution strategy**:

1. **Preserve local customizations**: Keep your resource limits, custom configs
2. **Adopt upstream fixes**: Take bug fixes and security patches
3. **Review breaking changes**: Carefully evaluate major version upgrades
4. **Test thoroughly**: Deploy to staging first

**Example conflict resolution**:
```yaml
# apps/postgres/deployment.yaml
<<<<<<< HEAD
# Our production tuning
resources:
  limits:
    cpu: "4000m"
    memory: "16Gi"
  requests:
    cpu: "2000m"
    memory: "8Gi"
=======
# Wild Directory new defaults
resources:
  limits:
    cpu: "2000m"
    memory: "8Gi"
  requests:
    cpu: "1000m"
    memory: "4Gi"
>>>>>>> wild-directory/main

# Resolution: Keep our production values
resources:
  limits:
    cpu: "4000m"
    memory: "16Gi"
  requests:
    cpu: "2000m"
    memory: "8Gi"
```

### Commit Message Conventions

**Format**: `<type>(<scope>): <subject>`

**Types**:
- `feat`: New app or feature
- `fix`: Bug fix or correction
- `config`: Configuration change
- `scale`: Resource scaling
- `upgrade`: Version upgrade
- `security`: Security-related change
- `docs`: Documentation change

**Examples**:
```bash
git commit -m "feat(redis): Add Redis cache for session storage"
git commit -m "scale(postgres): Increase CPU limits for production load"
git commit -m "fix(ghost): Correct domain configuration for SSL"
git commit -m "upgrade(immich): Update to v1.2.0 with new ML features"
git commit -m "security(all): Rotate database passwords"
git commit -m "config(mastodon): Enable SMTP for email notifications"
```

### Rollback Procedures

**Rollback entire app configuration**:
```bash
# Find commit to rollback to
git log --oneline -- apps/myapp/

# Revert specific commit
git revert abc123

# Or rollback to specific point
git checkout abc123 -- apps/myapp/
git commit -m "rollback(myapp): Revert to stable configuration"

# Deploy reverted state
wild app deploy myapp
```

**Emergency rollback (production incident)**:
```bash
# Immediately revert to last known good state
git log --oneline -5
git reset --hard abc123  # Last working commit
wild app deploy myapp

# Document the incident
git commit --allow-empty -m "emergency: Rolled back myapp due to production incident"
git push --force origin main  # Force push to update remote
```

### Collaboration Patterns

**Multiple admins working on same cluster**:

```bash
# Always pull before making changes
git pull origin main

# Use descriptive branch names
git checkout -b alice/add-monitoring
git checkout -b bob/upgrade-postgres

# Push branches for review
git push origin alice/add-monitoring

# Use PRs/MRs for review before merging to main
# This prevents conflicts and ensures peer review
```

**Code review checklist**:
- [ ] Changes tested in non-production environment
- [ ] Resource limits appropriate for workload
- [ ] Secrets not committed
- [ ] Dependencies deployed (if new app)
- [ ] Commit message follows conventions
- [ ] Breaking changes documented

### Backup and Disaster Recovery

**Regular backups**:
```bash
# Create tagged backup of current state
git tag -a backup-$(date +%Y%m%d) -m "Daily backup"
git push origin backup-$(date +%Y%m%d)

# Automated daily backup (cron)
0 2 * * * cd /var/lib/wild-central/instances/my-cloud && git tag backup-$(date +%Y%m%d-%H%M) && git push origin --tags
```

**Disaster recovery**:
```bash
# Clone instance repository to new Wild Central device
git clone git@github.com:myorg/wild-cloud-instances.git /var/lib/wild-central/instances/my-cloud

# Restore secrets from secure backup (NOT in git)
# (From password manager, Vault, encrypted backup, etc.)
cp ~/secure-backup/secrets.yaml /var/lib/wild-central/instances/my-cloud/

# Deploy all apps
cd /var/lib/wild-central/instances/my-cloud
for app in apps/*/; do
  wild app deploy $(basename $app)
done
```

### Git Workflow vs Web UI

**When git is better**:
- Complex changes requiring review
- Multi-app updates
- Compliance/audit requirements
- Team collaboration
- Emergency rollbacks

**When Web UI is better**:
- Quick configuration tweaks
- Adding single app
- Viewing current state
- Non-technical team members

**Hybrid approach**: Advanced users can use git for complex changes, Web UI for quick operations. The two workflows coexist peacefully since both modify the same instance directory.

## Conclusion

Wild Cloud's app lifecycle management spans three independent systems with different consistency guarantees. By understanding these systems and their boundaries, we can design operations that are:

- **Reliable**: Handle edge cases automatically
- **Simple**: Two-phase operations (cluster + local)
- **Safe**: Idempotent and recoverable
- **Fast**: Don't wait unnecessarily for eventual consistency

Additionally, for advanced users, the git-based workflow provides:
- **Auditable**: Full version control history
- **Collaborative**: Standard git workflows for team management
- **Recoverable**: Git revert/rollback capabilities
- **Professional**: Infrastructure-as-code best practices

The key insight is accepting eventual consistency at system boundaries while maintaining immediate consistency within each system. This allows operations to complete quickly for users while ensuring the system eventually reaches a consistent state.
