# Wild Cloud Service Management

This document explains the architecture and design of Wild Cloud's service installation system, including how it supports GitOps workflows.

## Service Installation State Machine

Wild Cloud uses a **multi-phase state machine** for service installation. This design separates concerns and enables GitOps workflows by allowing operators to control each phase independently.

### State Machine Phases

```
Available → Fetch → Validate Config → Compile → Deploy → Deployed
    ↑         ↓                           ↓         ↓
    └─────────┴───────────────────────────┴─────────┘
         (Can re-run any phase independently)
```

### Phase 1: Fetch

**Purpose**: Copy service files from embedded Wild Directory to instance directory.

**Files copied**:
- `wild-manifest.yaml` - Service metadata and configuration schema
- `README.md` - Documentation (optional)
- `install.sh` - Deployment script (optional)
- `kustomize.template/*` - Templated Kubernetes manifests (optional)

**Control**: `fetch` boolean flag
- `fetch=true`: Always fetch fresh files (overwrites existing)
- `fetch=false`: Only fetch if files don't exist (uses cached files)

**Result**: Service files exist in instance directory at:
```
instances/{instance}/setup/cluster-services/{service}/
├── wild-manifest.yaml
├── README.md
├── install.sh
└── kustomize.template/
```

**API Endpoint**: `POST /api/v1/instances/{instance}/services/{service}/fetch`

### Phase 2: Validate Configuration

**Purpose**: Ensure all required configuration is present before proceeding.

**Checks**:
- `configReferences` - Required config keys from other parts of the system
- `serviceConfig` - Service-specific configuration that needs values

**Behavior**: Fails install if configuration incomplete, prompting user to provide missing values.

**No separate API endpoint** - happens automatically during Install operation.

### Phase 3: Compile

**Purpose**: Process gomplate templates with instance config and secrets to generate deployment-ready Kubernetes manifests.

**Process**:
1. Read templates from `kustomize.template/` directory
2. Load `config.yaml` and `secrets.yaml` from instance directory
3. Process templates with gomplate
4. Output compiled manifests to `kustomize/` directory

**Result**: Ready-to-deploy Kubernetes manifests at:
```
instances/{instance}/setup/cluster-services/{service}/
└── kustomize/
    ├── kustomization.yaml
    ├── deployment.yaml
    ├── service.yaml
    └── ...
```

**API Endpoint**: `POST /api/v1/instances/{instance}/services/{service}/compile`

### Phase 4: Deploy

**Purpose**: Apply compiled Kubernetes manifests to the cluster.

**Method**:
- Runs `install.sh` script (if present) with `KUBECONFIG` environment variable set
- For services without `install.sh`, uses `kubectl apply -k kustomize/`

**Control**: `deploy` boolean flag
- `deploy=true`: Apply manifests to cluster
- `deploy=false`: Stop after Compile (prepare but don't deploy)

**Result**: Service running in Kubernetes cluster.

**API Endpoint**: `POST /api/v1/instances/{instance}/services/{service}/deploy`

### Install Operation (Orchestration)

The Install operation runs all phases in sequence with configurable control over Fetch and Deploy phases.

**API Endpoint**:
```
POST /api/v1/instances/{instance}/services
Body: {
  "name": "service-name",
  "fetch": boolean,   // Optional, defaults to false
  "deploy": boolean   // Optional, defaults to false
}
```

**Behavior**:
- Always runs: Validate → Compile
- Conditionally runs: Fetch (if `fetch=true` or files don't exist), Deploy (if `deploy=true`)

**CLI Examples**:
```bash
# Full install with defaults (use cache, compile and deploy)
wild service install metallb

# Fresh fetch and deploy
wild service install metallb --fetch

# Configure only, don't deploy yet
wild service install metallb --no-deploy

# Fresh fetch, configure only
wild service install metallb --fetch --no-deploy
```

## GitOps Integration

Wild Cloud is designed around **Infrastructure-as-Code** and **GitOps** principles. The entire instance configuration lives in `WILD_API_DATA_DIR`, which is intended to be a Git repository.

### What Gets Tracked in Git

**Tracked (committed to Git)**:
```
instances/{instance}/
├── config.yaml                    # Instance configuration
├── setup/
│   └── cluster-services/
│       └── {service}/
│           ├── wild-manifest.yaml      # Service metadata
│           ├── README.md                # Documentation
│           ├── install.sh               # Deployment script
│           ├── kustomize.template/      # Template source
│           └── kustomize/               # Compiled manifests ✓
└── apps/
    └── {app}/
        ├── manifest.yaml
        ├── kustomization.yaml
        └── *.yaml
```

**Ignored (not tracked)**:
```
.gitignore contains:
- secrets.yaml               # Contains sensitive data
- kubeconfig                 # Cluster credentials
- talosconfig                # Node credentials
- operations/                # Ephemeral operation logs
- backups/                   # Backup data
```

### Why Compiled Manifests Are Tracked

**Key Design Decision**: Both `kustomize.template/` AND `kustomize/` are tracked in Git.

**Rationale**:
1. **Auditability**: See exact manifests deployed at any point in history
2. **Diff-ability**: `git diff` shows actual Kubernetes resource changes, not just template changes
3. **Reproducibility**: Can re-apply exact same manifests without re-compiling
4. **Transparency**: No hidden template processing - what you see in Git is what runs in cluster
5. **GitOps reconciliation**: Standard GitOps tools can watch `kustomize/` directories

**Example Git History**:
```bash
$ git log --oneline instances/my-cloud/setup/cluster-services/traefik/

21246dc Update Traefik load balancer IP to 192.168.8.80
fa67e2b Configure Traefik with new TLS certificate
00558fc Add Traefik ingress controller
```

Each commit shows:
- Changes to `config.yaml` (configuration intent)
- Changes to `kustomize/` (actual manifests that were applied)

### How State Machine Phases Support GitOps

The multi-phase design **separates configuration from deployment** to enable GitOps workflows:

#### Fetch Phase → Version Control Integration
```bash
# Pull latest service definitions
wild service install traefik --fetch --no-deploy
```
- Service files become part of your Git repo
- Review changes before committing:
  ```bash
  git diff instances/my-cloud/setup/cluster-services/traefik/
  git commit -m "Update traefik to v3.4"
  ```

#### Compile Phase → Configuration as Code
```bash
# Generate manifests from configuration
wild service install traefik --no-deploy
```
- Reads `config.yaml` (tracked in Git)
- Processes templates
- Outputs to `kustomize/` (tracked in Git)
- **Committed manifests = declarative desired state**

#### Deploy Phase → Apply Desired State
```bash
# Apply to cluster
wild service install traefik
```
- Reads `kustomize/` from Git
- Applies to Kubernetes
- Cluster converges to Git state

### The `fetch` and `deploy` Flags

These flags enable GitOps workflows by controlling different aspects of the installation:

**`fetch` flag** → Controls *source* of manifests:
- `false`: Use existing files from Git (cached, version-controlled)
- `true`: Pull fresh from Wild Directory (upstream updates)

**`deploy` flag** → Controls *application* to cluster:
- `false`: Generate manifests, don't apply (prepare for Git commit and review)
- `true`: Apply manifests to cluster (convergence)

**GitOps Pattern**:
```bash
# Configuration change workflow
fetch=false, deploy=false  # Recompile with new config
# → git diff                # Review changes
# → git commit              # Record decision
# → deploy=true             # Apply to cluster

# Upstream update workflow
fetch=true, deploy=false   # Fetch new version
# → git diff                # Review changes
# → git commit              # Record decision
# → deploy=true             # Apply to cluster
```

### GitOps Workflows Enabled

#### Workflow 1: Review Before Deploy
```bash
# 1. Fetch and compile (don't deploy yet)
wild service install metallb --fetch --no-deploy

# 2. Review changes
cd $WILD_API_DATA_DIR
git diff

# 3. Commit if satisfied
git add instances/my-cloud/setup/cluster-services/metallb/
git commit -m "Add MetalLB load balancer"

# 4. Deploy when ready
wild service install metallb
```

#### Workflow 2: Configuration Changes
```bash
# 1. Edit config directly
vim instances/my-cloud/config.yaml

# 2. Recompile templates
wild service install traefik --no-deploy

# 3. Review generated manifests
git diff instances/my-cloud/setup/cluster-services/traefik/kustomize/

# 4. Commit and deploy
git commit -am "Update traefik ingress config"
wild service install traefik
```

#### Workflow 3: Multi-Instance Management
```yaml
# Same template, different configs
instances/
├── prod-cloud/
│   ├── config.yaml          # domain: prod.example.com
│   └── setup/cluster-services/traefik/kustomize/
└── staging-cloud/
    ├── config.yaml          # domain: staging.example.com
    └── setup/cluster-services/traefik/kustomize/
```

```bash
# Generate manifests for both
for instance in prod-cloud staging-cloud; do
  wild instance use $instance
  wild service install traefik --no-deploy
done

# Review all changes
git diff

# Deploy to staging first
wild instance use staging-cloud
wild service install traefik

# Test, then deploy to prod
wild instance use prod-cloud
wild service install traefik
```

#### Workflow 4: Disaster Recovery
```bash
# 1. Clone your wild-data repo to fresh system
git clone https://git.example.com/my-wild-cloud.git /var/lib/wild-central

# 2. Bootstrap new cluster
talosctl bootstrap --nodes <node-ip>

# 3. Deploy everything from Git
for service in $(ls instances/my-cloud/setup/cluster-services/); do
  wild service install $service
done

# All services restored from version-controlled state
```

### State Machine + GitOps = Declarative Infrastructure

```
Git Repository (Desired State)           Kubernetes Cluster (Actual State)
       ↓                                          ↑
  [config.yaml] ──gomplate──> [kustomize/] ──kubectl apply──> [Running Pods]
       ↓                            ↓
  [git commit]                 [git commit]
       ↓                            ↓
  [Auditability]              [Reproducibility]
```

The state machine allows:
1. **Fetch**: Update source of truth
2. **Compile**: Generate desired state
3. **Review**: Inspect changes before applying
4. **Commit**: Record decision in Git
5. **Deploy**: Converge cluster to desired state

This is **pure GitOps**: Git contains source of truth, cluster is reconciled to match.

### Comparison to Traditional GitOps Tools

**ArgoCD/FluxCD approach**:
- Watch Git repo → Auto-sync to cluster
- Requires separate Git repo for manifests
- Limited operator control over sync timing
- Continuous reconciliation

**Wild Cloud approach**:
- Git repo contains manifests
- Operator explicitly controls sync (deploy phase)
- Operator reviews and tests before syncing
- Same repo for all instances
- Manual reconciliation

Wild Cloud implements **"GitOps with manual sync"** - you maintain declarative config in Git, but retain control over when changes are applied. This is ideal for personal cloud environments where you want to review changes before deployment.

### Key Design Principles

1. **Separation of Concerns**: Each phase has a single, well-defined responsibility
2. **Explicit Control**: Operator controls which phases run via flags
3. **Git as Source of Truth**: All configuration and manifests tracked in Git
4. **Auditability**: Full history of what was deployed and when
5. **Reproducibility**: Can recreate cluster from Git repo alone
6. **Transparency**: Compiled manifests visible, no hidden transformations
7. **Flexibility**: Supports both automated and manual workflows
