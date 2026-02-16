# Wild Cloud App Lifecycle

App state spans three independent systems with different consistency models:

| System | Location | Consistency | Purpose |
|--------|----------|-------------|---------|
| Wild Directory | `wild-directory/{app}/` | Immutable, read-only | Shared templates |
| Instance Data | `data-dir/instances/{instance}/` | Immediate (file locks) | Config + compiled manifests |
| Kubernetes | API/etcd | Eventually consistent | Running workloads |

## User Workflows

**Advanced users (git-based)**: Instance data dir is a git repo. Direct manifest editing, Wild Directory tracked as upstream remote, standard git merge for updates. Full IaC workflow.

**Regular users (Web UI)**: Config changes via forms only, no manifest editing, automatic Wild Directory merges. Changes through config.yaml/secrets.yaml only.

| Aspect | Git Users | Web UI Users |
|--------|-----------|--------------|
| Manifest editing | Direct | Via config forms only |
| Wild Directory updates | Manual git merge | Automatic merge + review |
| Drift | Intentional (git-tracked) | Unintentional (reconcile via redeploy) |
| Rollback | `git revert` | Re-deploy previous state |

## State Components

### Wild Directory (Templates)

```
wild-directory/{app}/
├── manifest.yaml          # Metadata, dependencies, defaults
├── kustomization.yaml     # Kustomize config
├── deployment.yaml        # K8s workload (gomplate template)
├── service.yaml, ingress.yaml, namespace.yaml, pvc.yaml
├── db-init-job.yaml       # Optional
└── README.md
```

Template variables in `manifest.yaml`: `{{ .cloud.* }}`, `{{ .cluster.* }}`, `{{ .operator.* }}`, `{{ .app.* }}`, `{{ .apps.<ref>.* }}`, `{{ .secrets.* }}` (defaultSecrets only).

Template variables in resource `*.yaml` files: `{{ .* }}` resolves only from this app's defaultConfig. No access to secrets, cluster config, or other apps.

### Instance Data (Local State)

```
instances/{instance}/
├── config.yaml            # apps.{app}.* from defaultConfig
├── secrets.yaml           # apps.{app}.* generated + copied from deps
├── kubeconfig
├── apps/{app}/
│   ├── manifest.yaml      # Copy with installedAs mappings
│   └── *.yaml             # Compiled manifests (variables resolved)
└── operations/            # op_{action}_app_{app}_{timestamp}.json
```

### Kubernetes (Runtime State)

Per-app namespace containing: Deployment, ReplicaSet, Pods, Service, Ingress, PVC, Secret (`{app}-secrets`), optional ConfigMap and Job.

Labels applied via Kustomize `includeSelectors: true`:
```yaml
labels:
  - includeSelectors: true
    pairs:
      app: {app}
      managedBy: kustomize
      partOf: wild-cloud
```
Use simple component labels (`component: web`), not Helm-style (`app.kubernetes.io/name`).

### External Systems (Controller-Managed)

**External DNS**: Watches Ingress annotations (`external-dns.alpha.kubernetes.io/target`), creates DNS records at provider (Cloudflare). Propagation: 30s-5min.

**TLS Certificates**: Two shared wildcard certs avoid rate limits:
- `wildcard-wild-cloud-tls` (public: `*.cloud.example.com`)
- `wildcard-internal-wild-cloud-tls` (internal: `*.internal.cloud.example.com`)

Both live in `cert-manager` namespace. Deploy copies referenced certs to app namespace:
```go
wildcardSecrets := []string{"wildcard-wild-cloud-tls", "wildcard-internal-wild-cloud-tls"}
for _, secretName := range wildcardSecrets {
    if bytes.Contains(ingressContent, []byte(secretName)) {
        utilities.CopySecretBetweenNamespaces(kubeconfigPath, secretName, "cert-manager", appName)
    }
}
```

**MetalLB**: Assigns LoadBalancer IPs from pool, announces via ARP/BGP. ARP propagation: 0-60s.

**Deployment cascade**: kubectl apply -> Namespace+Resources -> external-dns (DNS 30s-5min) + cert-manager (TLS 30s-2min) + MetalLB (ARP 0-60s). Worst case: 5-7 min to fully operational.

## Lifecycle States

### NOT_ADDED (State 0)

Templates exist in Wild Directory. No instance or cluster state.

### ADDED (State 1)

After `wild app add {app}`:
1. Read manifest, resolve gomplate variables with instance config
2. Generate random secrets for `defaultSecrets` (if no default)
3. Copy secrets from dependencies for `requiredSecrets`
4. Compile templates -> write to `instance/apps/{app}/`
5. Append to `config.yaml` and `secrets.yaml` (file-locked)

Result: Local state consistent (config + secrets + compiled manifests). Cluster empty. Idempotent (re-add overwrites).

### DEPLOYING (State 2)

During `wild app deploy {app}`:
1. Pre-flight: check namespace status (must be Active or NotFound, not Terminating)
2. Create/update namespace (idempotent)
3. Create K8s secret from `secrets.yaml` (overwrite if exists)
4. Copy dependency secrets and TLS certs
5. `kubectl apply -k instance/apps/{app}/`

### DEPLOYED (State 3)

All resources running: pods healthy, services have endpoints, DNS created, TLS valid. Local state matches cluster intent.

### UPDATING Variants

**Config/secret changes (3a)**: Edit config.yaml/secrets.yaml -> redeploy. Config changes trigger template recompilation + rolling update. Secret-only changes update K8s secret but may need manual pod restart (`kubectl rollout restart`).

**Manifest changes (3b)**:
- *Git users*: Edit manifests directly, commit, deploy. `.package/` preserves originals.
- *Web UI users*: Only modify via config/secrets forms. System recompiles templates.

**Wild Directory update (3c)**:
- *Git users*: `git fetch wild-directory && git merge wild-directory/main`, resolve conflicts, deploy.
- *Web UI users*: System detects version diff via `.package/` comparison or manifest version field. Safe update: backup state, re-add from Wild Directory, merge new defaultConfig with existing config (new fields added, existing preserved), preserve secrets, recompile. User reviews diff then deploys. Destructive update: delete + re-add (loses customizations).

**Drift (3d)**:
- *Git users*: Intentional drift is IaC in action (tracked in git, not really drift).
- *Web UI users*: Any divergence is unintentional. Fix by redeploying (`kubectl apply` is idempotent, three-way merge).

### DELETING (State 4)

Two-phase delete:
1. **Cluster** (best effort): `kubectl delete namespace {app} --timeout=30s`. If stuck, force-remove finalizers.
2. **Local** (always succeeds): Remove `apps/{app}/`, delete entries from config.yaml and secrets.yaml.

Don't wait indefinitely for cluster cleanup. Accept temporary cross-system inconsistency.

### DELETED (State 5)

No local or cluster state. App can be re-added cleanly.

### STUCK_TERMINATING (Edge Case)

Namespace won't delete (custom finalizers, webhooks, network issues). Handled automatically by deploy pre-flight: detects Terminating, force-removes finalizers, retries up to 3 times with 5s waits.

## Design Principles

1. **Eventual consistency at boundaries**: Cluster and local state may temporarily diverge. Operations handle any state.
2. **Local state = source of truth**: config.yaml/secrets.yaml is authoritative. Cluster reflects current state.
3. **Idempotent everything**: `kubectl apply` not `create`, `kubectl delete --ignore-not-found`, `CREATE IF NOT EXISTS`.
4. **Fail forward**: Partial completion + retry = progress (not restart).
5. **No indefinite waits**: Operations timeout explicitly.
6. **No manual intervention**: Auto-recovery from stuck namespaces, etc.

All operations are idempotent: add (overwrites local), deploy (`kubectl apply`), delete (`--ignore-not-found`).

**Non-idempotent danger zones**: Secret regeneration (breaks running apps - only generate if key missing), DB init jobs (use idempotent SQL), finalizer removal (last resort only).

## Edge Cases

| Case | Behavior |
|------|----------|
| Namespace stuck terminating | Deploy pre-flight auto-cleans finalizers, retries 3x |
| Concurrent delete + deploy | Delete doesn't block on cluster; deploy handles Terminating namespace |
| Dependency not deployed | Deploy succeeds but pods CrashLoopBackOff. Future: pre-flight dep check |
| Secrets out of sync (K8s vs secrets.yaml) | Next deploy overwrites K8s secret. Always update secrets.yaml as source of truth |
| PVC retention after delete | Depends on ReclaimPolicy. Retain = data orphaned on re-add |
| Local state corrupted | `wild app add {app}` regenerates from Wild Directory |
| Secrets lost | Must re-add app (regenerates new secrets, apps need reconfiguration) |

## App Package Best Practices

**Security**: All pods require security contexts (runAsNonRoot, seccompProfile: RuntimeDefault, drop ALL capabilities). Common UID: 999 (PostgreSQL/Redis).

**DB init jobs**: Include `db-init-job.yaml` with idempotent SQL (`CREATE IF NOT EXISTS`). Use `restartPolicy: OnFailure`.

**DB URL secrets**: Never use runtime `$(VAR)` substitution in Kustomize. Use secretKeyRef pointing to `dbUrl` key defined in manifest's `defaultSecrets`:
```yaml
defaultSecrets:
  - key: dbUrl
    default: "postgres://{{ .app.dbUser }}:{{ .secrets.dbPassword }}@{{ .app.dbHost }}/{{ .app.dbName }}"
```

**External DNS**: Ingresses include `external-dns.alpha.kubernetes.io/target: {{ .domain }}` annotation.

**Helm conversion**: `helm template` -> add namespace.yaml -> `kustomize create --autodetect` -> create manifest.yaml -> replace values with gomplate vars -> replace Helm labels with Wild Cloud labels -> add security contexts + external-dns annotations.

## Git Workflow (Advanced Users)

Instance data dir as git repo. Wild Directory as upstream remote. `.gitignore` must exclude `secrets.yaml`.

**Branch strategy**: main (production), develop (staging), feature/* and hotfix/* branches.

**Wild Directory updates**: `git fetch wild-directory && git merge wild-directory/main`. Resolve conflicts preserving local customizations, adopting upstream fixes. Test before deploying.

**Commit format**: `<type>(<scope>): <subject>` where type is feat/fix/config/scale/upgrade/security/docs.

**Rollback**: `git revert <commit>` or `git checkout <commit> -- apps/{app}/` then redeploy.

**Disaster recovery**: Clone instance repo to new device, restore secrets.yaml from secure backup, deploy all apps.

**Hybrid**: Git for complex multi-app changes; Web UI for quick tweaks. Both modify same instance directory.

## Future Enhancements

- Pre-flight dependency validation (check required apps deployed before deploy)
- Periodic state reconciliation (background job comparing local vs cluster state)
- Built-in PVC backup/restore (`wild app backup/restore`)
- Dry-run mode (`wild app deploy --dry-run`)
