#!/bin/bash
set -e
set -o pipefail

# Ensure WILD_INSTANCE is set
if [ -z "${WILD_INSTANCE}" ]; then
    echo "❌ ERROR: WILD_INSTANCE is not set"
    exit 1
fi

# Ensure WILD_CENTRAL_DATA is set
if [ -z "${WILD_CENTRAL_DATA}" ]; then
    echo "❌ ERROR: WILD_CENTRAL_DATA is not set"
    exit 1
fi

# Ensure KUBECONFIG is set
if [ -z "${KUBECONFIG}" ]; then
    echo "❌ ERROR: KUBECONFIG is not set"
    exit 1
fi

INSTANCE_DIR="${WILD_CENTRAL_DATA}/instances/${WILD_INSTANCE}"
CLUSTER_SETUP_DIR="${INSTANCE_DIR}/setup/cluster-services"
CERT_MANAGER_DIR="${CLUSTER_SETUP_DIR}/cert-manager"

echo "🔧 === Setting up cert-manager ==="
echo ""

#######################
# Dependencies
#######################

# Check Traefik dependency
echo "🔍 Verifying Traefik is ready (required for cert-manager)..."
kubectl wait --for=condition=Available deployment/traefik -n traefik --timeout=60s 2>/dev/null || {
    echo "⚠️  Traefik not ready, but continuing with cert-manager installation"
    echo "💡 Note: cert-manager may not work properly without Traefik"
}

if [ ! -d "${CERT_MANAGER_DIR}/kustomize" ]; then
    echo "❌ ERROR: Compiled templates not found at ${CERT_MANAGER_DIR}/kustomize"
    echo "Templates should be compiled before deployment."
    exit 1
fi

# Note: DNS validation and Cloudflare token setup moved to configuration phase
# The configuration should be set via: wild config set cluster.certManager.cloudflare.*

########################
# Kubernetes components
########################

echo "📦 Installing cert-manager components..."
# Using stable URL for cert-manager installation
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.1/cert-manager.yaml || \
  kubectl apply -f https://github.com/jetstack/cert-manager/releases/download/v1.13.1/cert-manager.yaml

# Wait for cert-manager to be ready
echo "⏳ Waiting for cert-manager to be ready..."
kubectl wait --for=condition=Available deployment/cert-manager -n cert-manager --timeout=120s
kubectl wait --for=condition=Available deployment/cert-manager-cainjector -n cert-manager --timeout=120s
kubectl wait --for=condition=Available deployment/cert-manager-webhook -n cert-manager --timeout=120s

# Create Cloudflare API token secret
# Read token from Wild Central secrets file
echo "🔐 Creating Cloudflare API token secret..."
SECRETS_FILE="${WILD_CENTRAL_DATA}/instances/${WILD_INSTANCE}/secrets.yaml"
CLOUDFLARE_API_TOKEN=$(yq '.cloudflare.token' "$SECRETS_FILE" 2>/dev/null)

CLOUDFLARE_API_TOKEN=$(echo "$CLOUDFLARE_API_TOKEN")
if [ -z "$CLOUDFLARE_API_TOKEN" ] || [ "$CLOUDFLARE_API_TOKEN" = "null" ]; then
    echo "❌ ERROR: Cloudflare API token not found"
    echo "💡 Please set: wild secret set cloudflare.token YOUR_TOKEN"
    exit 1
fi

kubectl create secret generic cloudflare-api-token \
  --namespace cert-manager \
  --from-literal=api-token="${CLOUDFLARE_API_TOKEN}" \
  --dry-run=client -o yaml | kubectl apply -f -

# Ensure webhook is fully operational
echo "🔍 Verifying cert-manager webhook is fully operational..."
until kubectl get validatingwebhookconfigurations cert-manager-webhook &>/dev/null; do
    echo "⏳ Waiting for cert-manager webhook to register..."
    sleep 5
done

# Configure cert-manager to use external DNS for challenge verification
echo "🌐 Configuring cert-manager to use external DNS servers..."
kubectl patch deployment cert-manager -n cert-manager --patch '
spec:
  template:
    spec:
      dnsPolicy: None
      dnsConfig:
        nameservers:
          - "1.1.1.1"
          - "8.8.8.8"
        searches:
          - cert-manager.svc.cluster.local
          - svc.cluster.local
          - cluster.local
        options:
          - name: ndots
            value: "5"'

# Wait for cert-manager to restart with new DNS config
echo "⏳ Waiting for cert-manager to restart with new DNS configuration..."
kubectl rollout status deployment/cert-manager -n cert-manager --timeout=120s

########################
# Create issuers and certificates
########################

# Apply Let's Encrypt issuers and certificates using kustomize
echo "🚀 Creating Let's Encrypt issuers and certificates..."
kubectl apply -k ${CERT_MANAGER_DIR}/kustomize

# Wait for issuers to be ready
echo "⏳ Waiting for Let's Encrypt issuers to be ready..."
kubectl wait --for=condition=Ready clusterissuer/letsencrypt-prod --timeout=60s || echo "⚠️  Production issuer not ready, proceeding anyway..."
kubectl wait --for=condition=Ready clusterissuer/letsencrypt-staging --timeout=60s || echo "⚠️  Staging issuer not ready, proceeding anyway..."

# Give cert-manager a moment to process the certificates
sleep 5

######################################
# Fix stuck certificates and cleanup
######################################

needs_restart=false

# STEP 1: Fix certificates stuck with 404 errors
echo "🔍 Checking for certificates with failed issuance attempts..."
stuck_certs=$(kubectl get certificates --all-namespaces -o json 2>/dev/null | \
    jq -r '.items[] | select(.status.conditions[]? | select(.type=="Issuing" and .status=="False" and (.message | contains("404")))) | "\(.metadata.namespace) \(.metadata.name)"')

if [ -n "$stuck_certs" ]; then
    echo "⚠️  Found certificates stuck with non-existent orders, recreating them..."
    echo "$stuck_certs" | while read ns name; do
        echo "🔄 Recreating certificate $ns/$name..."
        cert_spec=$(kubectl get certificate "$name" -n "$ns" -o json | jq '.spec')
        kubectl delete certificate "$name" -n "$ns"
        echo "{\"apiVersion\":\"cert-manager.io/v1\",\"kind\":\"Certificate\",\"metadata\":{\"name\":\"$name\",\"namespace\":\"$ns\"},\"spec\":$cert_spec}" | kubectl apply -f -
    done
    needs_restart=true
    sleep 5
else
    echo "✅ No certificates stuck with failed orders"
fi

# STEP 2: Clean up orphaned orders
echo "🔍 Checking for orphaned ACME orders..."
orphaned_orders=$(kubectl logs -n cert-manager deployment/cert-manager --tail=200 2>/dev/null | \
    grep -E "failed to retrieve the ACME order.*404" 2>/dev/null | \
    sed -n 's/.*resource_name="\([^"]*\)".*/\1/p' | \
    sort -u || true)

if [ -n "$orphaned_orders" ]; then
    echo "⚠️  Found orphaned ACME orders from logs"
    for order in $orphaned_orders; do
        echo "🗑️  Deleting orphaned order: $order"
        orders_found=$(kubectl get orders --all-namespaces 2>/dev/null | grep "$order" 2>/dev/null || true)
        if [ -n "$orders_found" ]; then
            echo "$orders_found" | while read ns name rest; do
                kubectl delete order "$name" -n "$ns" 2>/dev/null || true
            done
        fi
    done
    needs_restart=true
else
    echo "✅ No orphaned orders found in logs"
fi

# STEP 2.5: Check for Cloudflare DNS cleanup errors
echo "🔍 Checking for Cloudflare DNS cleanup errors..."
cloudflare_errors=$(kubectl logs -n cert-manager deployment/cert-manager --tail=200 2>/dev/null | \
    grep -c "Error: 7003.*Could not route" 2>/dev/null || echo "0")

if [ "$cloudflare_errors" -gt "0" ]; then
    echo "⚠️  Found $cloudflare_errors Cloudflare DNS cleanup errors (stale DNS record references)"
    echo "💡 Deleting stuck challenges and orders to allow fresh start"

    # Delete all challenges and orders in cert-manager namespace
    kubectl delete challenges --all -n cert-manager 2>/dev/null || true
    kubectl delete orders --all -n cert-manager 2>/dev/null || true

    needs_restart=true
else
    echo "✅ No Cloudflare DNS cleanup errors"
fi

# STEP 3: Single restart if anything needs cleaning
if [ "$needs_restart" = true ]; then
    echo "🔄 Restarting cert-manager to clear internal state..."
    kubectl rollout restart deployment cert-manager -n cert-manager
    kubectl rollout status deployment/cert-manager -n cert-manager --timeout=120s
    echo "⏳ Waiting for cert-manager to recreate fresh challenges..."
    sleep 15
else
    echo "✅ No restart needed - cert-manager state is clean"
fi

#########################
# Final checks
#########################

# Wait for the certificates to be issued with progress feedback
echo "⏳ Waiting for wildcard certificates to be ready (this may take several minutes)..."

# Function to wait for certificate with progress output
wait_for_cert() {
    local cert_name="$1"
    local timeout=300
    local elapsed=0

    echo "  📜 Checking $cert_name..."

    while [ $elapsed -lt $timeout ]; do
        if kubectl get certificate "$cert_name" -n cert-manager -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}' 2>/dev/null | grep -q "True"; then
            echo "  ✅ $cert_name is ready"
            return 0
        fi

        # Show progress every 30 seconds
        if [ $((elapsed % 30)) -eq 0 ] && [ $elapsed -gt 0 ]; then
            local status=$(kubectl get certificate "$cert_name" -n cert-manager -o jsonpath='{.status.conditions[?(@.type=="Ready")].message}' 2>/dev/null || echo "Waiting...")
            echo "  ⏳ Still waiting for $cert_name... ($elapsed/${timeout}s) - $status"
        fi

        sleep 5
        elapsed=$((elapsed + 5))
    done

    echo "  ⚠️  Timeout waiting for $cert_name (will continue anyway)"
    return 1
}

wait_for_cert "wildcard-internal-wild-cloud"
wait_for_cert "wildcard-wild-cloud"

# Final health check
echo "🔍 Performing final cert-manager health check..."
failed_certs=$(kubectl get certificates --all-namespaces -o json 2>/dev/null | jq -r '.items[] | select(.status.conditions[]? | select(.type=="Ready" and .status!="True")) | "\(.metadata.namespace)/\(.metadata.name)"' | wc -l)
if [ "$failed_certs" -gt 0 ]; then
    echo "⚠️  Found $failed_certs certificates not in Ready state"
    echo "💡 Check certificate status with: kubectl get certificates --all-namespaces"
    echo "💡 Check cert-manager logs with: kubectl logs -n cert-manager deployment/cert-manager"
else
    echo "✅ All certificates are in Ready state"
fi

echo ""
echo "✅ cert-manager setup complete!"
echo ""
echo "💡 To verify the installation:"
echo "  kubectl get certificates --all-namespaces"
echo "  kubectl get clusterissuers"
