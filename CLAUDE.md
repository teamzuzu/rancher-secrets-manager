# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

A Rancher plugin that implements a secrets manager for Rancher's management cluster and distributes secrets to downstream clusters. It consists of two deliverables:

1. **Go controller** — a Kubernetes operator packaged as a Helm chart, deployed into the Rancher management cluster
2. **Rancher UI Extension** — a Vue.js plugin for the Rancher dashboard

## Commands

### Go Controller

```bash
# Build
make build                    # compiles cmd/manager to bin/manager
make docker-build             # builds the container image

# Generate CRD manifests and deepcopy code (run after changing types in pkg/api/)
make generate                 # runs controller-gen object/deepcopy
make manifests                # runs controller-gen crd/rbac/webhook

# Lint
make lint                     # golangci-lint run

# Test
make test                     # go test ./... with envtest
go test ./pkg/controller/...  # single package
go test -run TestManagedSecret ./pkg/controller/managedsecret/  # single test

# Run locally against current kubeconfig
make run
```

### Helm Chart

```bash
# Lint and render
helm lint charts/rancher-secrets-manager
helm template rancher-secrets-manager charts/rancher-secrets-manager --debug

# Install into management cluster (development)
helm install rancher-secrets-manager charts/rancher-secrets-manager \
  -n cattle-secrets-system --create-namespace

# Install CRDs only
kubectl apply -f charts/rancher-secrets-manager/crds/
```

### Dev / CI Environment

```bash
# Spin up local Rancher dev environment (management + 2 downstream clusters)
./scripts/setup-dev.sh

# Tear it down
./scripts/setup-dev.sh teardown

# Override defaults
RANCHER_VERSION=2.14.1 DOWNSTREAM_COUNT=1 ./scripts/setup-dev.sh
```

The script creates a shared Docker network (`rancher-dev`), starts k3d clusters on it, installs cert-manager + Rancher on the management cluster, then imports each downstream cluster via the Rancher v3 API. The cattle-cluster-agent on downstream clusters reaches Rancher using the management LB container's IP on the shared network, resolved via `nip.io`.

CI runs the same flow via `.github/workflows/e2e.yml` (GitHub Actions, triggers: PR + `workflow_dispatch`).

### UI Extension

```bash
cd ui
yarn install
yarn dev          # dev server with HMR against a real Rancher instance
yarn build        # production build
yarn lint         # eslint
```

## Architecture

### Custom Resource

`ManagedSecret` (`secrets.cattle.io/v1alpha1`) is the central CRD. It lives in the management cluster and describes:
- A `secretRef` pointing to a source Kubernetes `Secret` in the management cluster
- A `targets` list where each entry specifies a downstream cluster (by name or label selector) **and** the namespace to create the secret in on that cluster

The controller copies the source secret's `data` into a native `Secret` on each targeted downstream cluster and tracks sync state per-cluster in `.status.syncStatus`.

### How the controller reaches downstream clusters

The controller does **not** manage its own cluster credentials. Instead it proxies through Rancher's built-in API proxy (backed by `cattle-cluster-agent` connections). From inside the management cluster the proxy is reachable at:

```
https://<rancher-service>/k8s/clusters/<cluster-id>/
```

**Authentication:** Rancher's proxy rejects plain Kubernetes ServiceAccount tokens — it requires a Rancher user token. The controller reads the token from the Fleet-managed kubeconfig secret at `fleet-default/<clusterID>-kubeconfig` (populated automatically by Rancher when a cluster is imported). If that secret doesn't exist (e.g. non-Fleet clusters), it falls back to the controller's SA token.

**TLS:** `InsecureSkipVerify: true` is set for all Rancher proxy connections. Rancher's `dynamiclistener` CA uses an ECDSA encoding that Go 1.22+'s strict x509 verification rejects — even when the cert chain is structurally valid. Since this is internal cluster traffic to a known service endpoint, skipping verification is acceptable.

The `pkg/rancher/` package wraps this: it lists `management.cattle.io/v3 Cluster` objects to resolve cluster selectors, then builds a per-cluster `rest.Config` via `configFromFleetKubeconfig` (or falls back to SA token).

### Controller reconcile loop

```
ManagedSecret changed
  → resolve target clusters (list Clusters, apply selector)
  → for each (cluster, namespace):
      → build downstream rest.Config (Rancher proxy)
      → create or update the Secret in that namespace
      → record result in status.syncStatus[cluster]
  → update ManagedSecret status
```

The controller also watches the referenced source `Secret` and re-queues any `ManagedSecret` that references it when it changes.

### Helm chart layout

```
charts/rancher-secrets-manager/
├── crds/               # CRD YAML (installed before templates)
├── templates/
│   ├── deployment.yaml
│   ├── serviceaccount.yaml
│   ├── clusterrole.yaml        # get/list/watch on Clusters, Secrets (cluster-wide, covers fleet-default)
│   └── clusterrolebinding.yaml
└── values.yaml
```

The controller runs in the `cattle-secrets-system` namespace by default.

### UI Extension

Built with the [Rancher Extension SDK](https://extensions.rancher.io). The extension registers a top-level nav item "Secrets Manager" and provides:
- A list view of `ManagedSecret` resources with per-cluster sync status badges
- A detail/edit view with the target cluster/namespace matrix
- A create form

The extension talks to the management cluster's Rancher API (Steve) directly — no separate backend. CRD resources are available via the Steve REST API once the Helm chart is installed.

### Key dependency versions

| Dependency | Version |
|---|---|
| Go | 1.22+ |
| controller-runtime | v0.18+ |
| Rancher API group | `management.cattle.io/v3` |
| Rancher target | 2.9+ |
| Node / Vue | 20 / 3 |
| Rancher Extension SDK | v2 |
