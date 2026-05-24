#!/usr/bin/env bash
# Sets up a local Rancher dev environment: 1 management cluster + N downstream clusters.
# Mirrors the CI setup exactly so local behaviour matches CI.
#
# Usage:
#   ./hack/setup-dev.sh            # create everything
#   DOWNSTREAM_COUNT=1 ./hack/setup-dev.sh
#   ./hack/setup-dev.sh teardown   # destroy all clusters and the Docker network

set -euo pipefail

RANCHER_VERSION="${RANCHER_VERSION:-2.14.1}"
CERT_MANAGER_VERSION="${CERT_MANAGER_VERSION:-v1.16.3}"
DOWNSTREAM_COUNT="${DOWNSTREAM_COUNT:-2}"
NETWORK="rancher-dev"
MANAGEMENT="management"
RANCHER_BOOTSTRAP_PASSWORD="adminadmin"   # >=12 chars required by Rancher 2.8+

# ── helpers ──────────────────────────────────────────────────────────────────

log()  { echo "▶ $*"; }
die()  { echo "✗ $*" >&2; exit 1; }

need() {
  for cmd in "$@"; do
    command -v "$cmd" &>/dev/null || die "Required tool not found: $cmd"
  done
}

wait_rollout() {
  local ctx="$1" ns="$2" deploy="$3"
  log "Waiting for $deploy in $ns ($ctx)..."
  kubectl --context "$ctx" -n "$ns" rollout status deployment/"$deploy" --timeout=300s
}

# ── teardown ─────────────────────────────────────────────────────────────────

teardown() {
  log "Tearing down dev environment..."
  for i in $(seq 1 "$DOWNSTREAM_COUNT"); do
    k3d cluster delete "downstream-${i}" 2>/dev/null || true
  done
  k3d cluster delete "$MANAGEMENT" 2>/dev/null || true
  docker network rm "$NETWORK" 2>/dev/null || true
  log "Done."
}

[[ "${1:-}" == "teardown" ]] && { teardown; exit 0; }

# ── preflight ────────────────────────────────────────────────────────────────

need docker k3d kubectl helm curl jq

# ── clusters ─────────────────────────────────────────────────────────────────

log "Creating Docker network: $NETWORK"
docker network inspect "$NETWORK" &>/dev/null || docker network create "$NETWORK"

log "Creating management cluster..."
# Keep traefik — Rancher's Ingress requires an ingress controller.
k3d cluster create "$MANAGEMENT" \
  --network "$NETWORK" \
  --k3s-arg "--disable=servicelb@server:*" \
  -p "80:80@loadbalancer" \
  -p "443:443@loadbalancer" \
  --wait

for i in $(seq 1 "$DOWNSTREAM_COUNT"); do
  log "Creating downstream-${i} cluster..."
  k3d cluster create "downstream-${i}" \
    --network "$NETWORK" \
    --k3s-arg "--disable=traefik,servicelb@server:*" \
    --wait
done

# ── resolve Rancher hostname ──────────────────────────────────────────────────
# The LB container's IP on the shared network is reachable from all other containers.

LB_IP=$(docker inspect "k3d-${MANAGEMENT}-serverlb" \
  --format "{{(index .NetworkSettings.Networks \"${NETWORK}\").IPAddress}}")
[[ -z "$LB_IP" ]] && die "Could not determine LB IP — did the management cluster start?"
RANCHER_HOST="rancher.${LB_IP//./-}.nip.io"
log "Rancher hostname: $RANCHER_HOST (LB IP: $LB_IP)"

MGMT_CTX="k3d-${MANAGEMENT}"

# ── cert-manager ─────────────────────────────────────────────────────────────

log "Installing cert-manager ${CERT_MANAGER_VERSION}..."
kubectl --context "$MGMT_CTX" apply \
  -f "https://github.com/cert-manager/cert-manager/releases/download/${CERT_MANAGER_VERSION}/cert-manager.yaml"
wait_rollout "$MGMT_CTX" cert-manager cert-manager
wait_rollout "$MGMT_CTX" cert-manager cert-manager-webhook

# ── Rancher ───────────────────────────────────────────────────────────────────

log "Installing Rancher ${RANCHER_VERSION}..."
helm repo add rancher-stable https://releases.rancher.com/server-charts/stable --force-update
helm repo update rancher-stable

helm upgrade --install rancher rancher-stable/rancher \
  --kube-context "$MGMT_CTX" \
  --namespace cattle-system \
  --create-namespace \
  --version "$RANCHER_VERSION" \
  --set hostname="$RANCHER_HOST" \
  --set bootstrapPassword="$RANCHER_BOOTSTRAP_PASSWORD" \
  --set replicas=1 \
  --set ingress.tls.source=rancher \
  --wait \
  --timeout 10m

log "Rancher is up at https://${RANCHER_HOST}"

# ── bootstrap token ───────────────────────────────────────────────────────────

log "Obtaining Rancher API token..."
RANCHER_URL="https://${RANCHER_HOST}"

# Retry until Rancher API is ready
for i in $(seq 1 30); do
  HTTP=$(curl -sk -o /dev/null -w "%{http_code}" "${RANCHER_URL}/ping") || true
  [[ "$HTTP" == "200" ]] && break
  sleep 5
done
[[ "$HTTP" != "200" ]] && die "Rancher API never became ready"

RANCHER_TOKEN=$(curl -sk -X POST "${RANCHER_URL}/v3-public/localProviders/local?action=login" \
  -H "Content-Type: application/json" \
  -d "{\"username\":\"admin\",\"password\":\"${RANCHER_BOOTSTRAP_PASSWORD}\"}" \
  | jq -r '.token')
[[ -z "$RANCHER_TOKEN" || "$RANCHER_TOKEN" == "null" ]] && die "Failed to obtain Rancher token"

# ── import downstream clusters ────────────────────────────────────────────────

for i in $(seq 1 "$DOWNSTREAM_COUNT"); do
  CLUSTER_NAME="downstream-${i}"
  DS_CTX="k3d-${CLUSTER_NAME}"
  log "Importing ${CLUSTER_NAME} into Rancher..."

  # Create the cluster object
  CLUSTER_ID=$(curl -sk -X POST "${RANCHER_URL}/v3/clusters" \
    -H "Authorization: Bearer ${RANCHER_TOKEN}" \
    -H "Content-Type: application/json" \
    -d "{\"type\":\"cluster\",\"name\":\"${CLUSTER_NAME}\"}" \
    | jq -r '.id')
  [[ -z "$CLUSTER_ID" || "$CLUSTER_ID" == "null" ]] && die "Failed to create cluster object for ${CLUSTER_NAME}"

  # Fetch registration manifest (poll until token is created)
  MANIFEST_URL=""
  for attempt in $(seq 1 20); do
    MANIFEST_URL=$(curl -sk \
      "${RANCHER_URL}/v3/clusterregistrationtokens?clusterId=${CLUSTER_ID}" \
      -H "Authorization: Bearer ${RANCHER_TOKEN}" \
      | jq -r '.data[0].manifestUrl // empty')
    [[ -n "$MANIFEST_URL" ]] && break
    sleep 3
  done
  [[ -z "$MANIFEST_URL" ]] && die "No registration manifest URL for ${CLUSTER_NAME}"

  # Apply the agent manifest to the downstream cluster
  # The manifest URL uses the Rancher hostname, which is reachable inside the shared Docker network
  kubectl --context "$DS_CTX" apply -f "$MANIFEST_URL"
  log "${CLUSTER_NAME}: cattle-cluster-agent applied, waiting for registration..."
done

# ── wait for clusters to become Active ────────────────────────────────────────

log "Waiting for all downstream clusters to become Active (this can take ~2 min)..."
for i in $(seq 1 "$DOWNSTREAM_COUNT"); do
  CLUSTER_NAME="downstream-${i}"
  for attempt in $(seq 1 60); do
    STATE=$(curl -sk "${RANCHER_URL}/v3/clusters?name=${CLUSTER_NAME}" \
      -H "Authorization: Bearer ${RANCHER_TOKEN}" \
      | jq -r '.data[0].state // empty')
    [[ "$STATE" == "active" ]] && { log "${CLUSTER_NAME}: Active"; break; }
    [[ $attempt -eq 60 ]] && die "${CLUSTER_NAME} never became Active (last state: $STATE)"
    sleep 5
  done
done

# ── summary ───────────────────────────────────────────────────────────────────

cat <<EOF

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  Dev environment ready
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  Rancher UI:    https://${RANCHER_HOST}
  Username:      admin
  Password:      ${RANCHER_BOOTSTRAP_PASSWORD}

  Management context:  ${MGMT_CTX}
  Downstream contexts: $(seq -s' ' 1 "$DOWNSTREAM_COUNT" | sed 's/\([0-9]*\)/k3d-downstream-\1/g')

  To teardown:   ./hack/setup-dev.sh teardown
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
EOF

# Export useful vars for subsequent scripts (source this file to pick them up)
echo "export RANCHER_URL=${RANCHER_URL}"
echo "export RANCHER_TOKEN=${RANCHER_TOKEN}"
echo "export MGMT_CTX=${MGMT_CTX}"
