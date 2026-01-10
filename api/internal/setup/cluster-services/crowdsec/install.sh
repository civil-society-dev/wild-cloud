#!/bin/bash
set -e
set -o pipefail

# Ensure WILD_INSTANCE is set
if [ -z "${WILD_INSTANCE}" ]; then
    echo "ERROR: WILD_INSTANCE is not set"
    exit 1
fi

# Ensure WILD_API_DATA_DIR is set
if [ -z "${WILD_API_DATA_DIR}" ]; then
    echo "ERROR: WILD_API_DATA_DIR is not set"
    exit 1
fi

# Ensure KUBECONFIG is set
if [ -z "${KUBECONFIG}" ]; then
    echo "ERROR: KUBECONFIG is not set"
    exit 1
fi

INSTANCE_DIR="${WILD_API_DATA_DIR}/instances/${WILD_INSTANCE}"
CLUSTER_SETUP_DIR="${INSTANCE_DIR}/setup/cluster-services"
CROWDSEC_DIR="${CLUSTER_SETUP_DIR}/crowdsec"
SECRETS_FILE="${INSTANCE_DIR}/secrets.yaml"

echo "=== Setting up CrowdSec Security Engine ==="
echo ""

# Check traefik dependency
echo "Verifying Traefik is ready (required for CrowdSec bouncer)..."
kubectl wait --for=condition=Available deployment/traefik -n traefik --timeout=60s 2>/dev/null || {
    echo "WARNING: Traefik not ready, but continuing with CrowdSec installation"
    echo "Note: CrowdSec bouncer will not work until Traefik is available"
}

# Templates should already be compiled
echo "Using pre-compiled CrowdSec templates..."
if [ ! -d "${CROWDSEC_DIR}/kustomize" ]; then
    echo "ERROR: Compiled templates not found at ${CROWDSEC_DIR}/kustomize"
    echo "Templates should be compiled before deployment."
    exit 1
fi

# Apply CrowdSec manifests using kustomize
echo "Deploying CrowdSec..."
kubectl apply -k ${CROWDSEC_DIR}/kustomize

# Setup CrowdSec agent secret
echo "Creating CrowdSec agent secret..."
AGENT_PASSWORD=$(yq '.cluster.crowdsec.agentPassword' "$SECRETS_FILE" 2>/dev/null | tr -d '"')

if [ -z "$AGENT_PASSWORD" ] || [ "$AGENT_PASSWORD" = "null" ]; then
    echo "Generating new agent password..."
    AGENT_PASSWORD=$(openssl rand -base64 32)
    # Note: The API should have already set this in secrets.yaml during config phase
    echo "WARNING: Agent password not found in secrets.yaml"
    echo "Using generated password - you may want to persist this"
fi

kubectl create secret generic crowdsec-agent-secret \
  --namespace crowdsec \
  --from-literal=password="${AGENT_PASSWORD}" \
  --dry-run=client -o yaml | kubectl apply -f -

# Wait for CrowdSec agent to be ready
echo "Waiting for CrowdSec agent to be ready..."
kubectl rollout status deployment/crowdsec -n crowdsec --timeout=120s

# Register bouncer with CrowdSec agent and create bouncer secret
echo "Registering bouncer with CrowdSec agent..."
BOUNCER_API_KEY=$(yq '.cluster.crowdsec.bouncerApiKey' "$SECRETS_FILE" 2>/dev/null | tr -d '"')

if [ -z "$BOUNCER_API_KEY" ] || [ "$BOUNCER_API_KEY" = "null" ]; then
    echo "Generating new bouncer API key from CrowdSec agent..."
    # Remove existing bouncer if it exists
    kubectl exec -n crowdsec deploy/crowdsec -- cscli bouncers delete traefik-bouncer 2>/dev/null || true
    # Add new bouncer and capture the key
    BOUNCER_API_KEY=$(kubectl exec -n crowdsec deploy/crowdsec -- cscli bouncers add traefik-bouncer -o raw)
    echo "Generated bouncer API key - you may want to persist this in secrets.yaml"
fi

kubectl create secret generic crowdsec-bouncer-secret \
  --namespace crowdsec \
  --from-literal=api-key="${BOUNCER_API_KEY}" \
  --dry-run=client -o yaml | kubectl apply -f -

# Restart bouncer to pick up the secret
echo "Restarting bouncer deployment..."
kubectl rollout restart deployment/traefik-crowdsec-bouncer -n crowdsec

# Wait for bouncer to be ready
echo "Waiting for CrowdSec bouncer to be ready..."
kubectl rollout status deployment/traefik-crowdsec-bouncer -n crowdsec --timeout=60s

# Patch Traefik to use CrowdSec middleware by default on websecure entrypoint
echo "Configuring Traefik to use CrowdSec security chain by default..."
kubectl patch deployment traefik -n traefik --type='json' -p='[
  {
    "op": "add",
    "path": "/spec/template/spec/containers/0/args/-",
    "value": "--entryPoints.websecure.http.middlewares=crowdsec-security-chain@kubernetescrd"
  }
]' 2>/dev/null || {
    echo "Note: Traefik may already have middleware configured or patch failed"
    echo "You can manually configure default middleware if needed"
}

echo ""
echo "CrowdSec installed successfully"
echo ""
echo "All ingresses are now protected by default with:"
echo "  - Threat detection (CrowdSec)"
echo "  - Rate limiting (100 req/min)"
echo "  - Security headers (HSTS, XSS protection, etc.)"
echo ""
echo "To verify the installation:"
echo "  kubectl get pods -n crowdsec"
echo "  kubectl exec -n crowdsec deploy/crowdsec -- cscli bouncers list"
echo "  kubectl exec -n crowdsec deploy/crowdsec -- cscli decisions list"
echo ""
echo "To opt-out a specific ingress from CrowdSec protection:"
echo "  Add annotation: traefik.ingress.kubernetes.io/router.middlewares: \"\""
echo ""
