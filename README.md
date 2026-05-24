# rancher-secrets-manager

A Rancher plugin that distributes secrets from the management cluster to downstream clusters. Define a `ManagedSecret` once; the controller keeps the secret in sync across every targeted cluster and namespace — no per-cluster credentials needed.

```
ManagedSecret → controller → Rancher API proxy → cattle-cluster-agent → downstream Secret
```

## Contents

- [How it works](#how-it-works)
- [Installation](#installation)
  - [Prerequisites](#prerequisites)
  - [1. Build and push the container image](#1-build-and-push-the-container-image)
  - [2. Install the Helm chart](#2-install-the-helm-chart)
  - [3. Verify the controller is running](#3-verify-the-controller-is-running)
  - [4. Install the UI extension (optional)](#4-install-the-ui-extension-optional)
- [Configuration reference](#configuration-reference)
- [Usage](#usage)
- [Uninstall](#uninstall)
- [Development](#development)

---

## How it works

1. A `ManagedSecret` (cluster-scoped CRD) references a source `Secret` in the management cluster and a list of targets — each target specifies a downstream cluster (by name or label selector) and a namespace.
2. The controller resolves targets by listing `management.cattle.io/v3 Cluster` objects.
3. For each `(cluster, namespace)` pair it creates or updates the secret by proxying through Rancher's built-in API proxy endpoint (`/k8s/clusters/<cluster-id>/`), using the controller's own ServiceAccount token — no extra credentials required on downstream clusters.
4. Sync state is tracked per-cluster in `.status.syncStatus`.

---

## Installation

### Prerequisites

| Requirement | Version |
|---|---|
| Rancher | 2.9 or later |
| Kubernetes (management cluster) | 1.26 or later |
| Helm | 3.x |
| kubectl | configured for the **management cluster** |
| Docker | for building the controller image |
| Container registry | accessible from the management cluster |

> **Important:** All `kubectl` and `helm` commands below must target the **management cluster** (the cluster where Rancher itself runs), not a downstream cluster.

---

### 1. Build and push the container image

Clone the repository and build the image, replacing `your-registry` with your container registry:

```bash
git clone https://github.com/teamzuzu/rancher-secrets-manager.git
cd rancher-secrets-manager

# Build the image
make docker-build IMG=your-registry/rancher-secrets-manager:0.1.0

# Push to your registry
make docker-push IMG=your-registry/rancher-secrets-manager:0.1.0
```

> If your registry requires authentication, run `docker login your-registry` before pushing.

---

### 2. Install the Helm chart

Install into the `cattle-secrets-system` namespace, pointing at the image you just pushed:

```bash
helm install rancher-secrets-manager charts/rancher-secrets-manager \
  --namespace cattle-secrets-system \
  --create-namespace \
  --set image.repository=your-registry/rancher-secrets-manager \
  --set image.tag=0.1.0
```

The default `rancher.url` (`https://rancher.cattle-system.svc`) works in most standard Rancher deployments. If your Rancher service is in a different namespace or uses a custom hostname, override it:

```bash
helm install rancher-secrets-manager charts/rancher-secrets-manager \
  --namespace cattle-secrets-system \
  --create-namespace \
  --set image.repository=your-registry/rancher-secrets-manager \
  --set image.tag=0.1.0 \
  --set rancher.url=https://rancher.example.com
```

**If Rancher uses a private CA**, mount the CA bundle:

```bash
# Create a ConfigMap with your CA certificate
kubectl create configmap rancher-ca \
  --namespace cattle-secrets-system \
  --from-file=ca.pem=/path/to/ca.pem

# Reference it in the chart (see Configuration reference for volume mount details)
helm install rancher-secrets-manager charts/rancher-secrets-manager \
  --namespace cattle-secrets-system \
  --create-namespace \
  --set image.repository=your-registry/rancher-secrets-manager \
  --set image.tag=0.1.0 \
  --set rancher.caBundle=/etc/ssl/rancher/ca.pem
```

---

### 3. Verify the controller is running

```bash
# Controller pod should be Running within ~30 seconds
kubectl get pods -n cattle-secrets-system

# Expected output:
# NAME                                         READY   STATUS    RESTARTS   AGE
# rancher-secrets-manager-6d8f9b7c4-xkp2j     1/1     Running   0          45s

# Check the controller logs for any startup errors
kubectl logs -n cattle-secrets-system \
  -l app.kubernetes.io/name=rancher-secrets-manager --tail=20
```

A healthy controller logs something like:

```
{"level":"info","msg":"starting manager"}
{"level":"info","msg":"starting server","kind":"health probe","addr":"[::]:8081"}
{"level":"info","msg":"Starting EventSource","controller":"managedsecret"}
{"level":"info","msg":"Starting Controller","controller":"managedsecret"}
```

**Confirm the CRD is installed:**

```bash
kubectl get crd managedsecrets.secrets.cattle.io
# NAME                              CREATED AT
# managedsecrets.secrets.cattle.io  2024-01-15T10:00:00Z
```

---

### 4. Install the UI extension (optional)

The UI extension adds a **Secrets Manager** section to the Rancher dashboard for managing `ManagedSecret` resources through a graphical interface.

#### Build the extension bundle

```bash
cd ui
yarn install
yarn build-pkg
# Built to: ui/dist-pkg/rancher-secrets-manager-0.1.0/
```

#### Load the extension in Rancher (developer mode)

1. Serve the extension bundle from the `dist-pkg/` directory:
   ```bash
   # From inside the ui/ directory
   yarn serve-pkgs
   # Serving on http://localhost:4500
   ```

2. In the Rancher dashboard, navigate to **☰ → Extensions → ⋮ → Developer Load**
3. Enter the extension URL: `http://<your-machine-ip>:4500/rancher-secrets-manager-0.1.0/rancher-secrets-manager-0.1.0.umd.min.js`
4. Click **Load**

The **Secrets Manager** item will appear in the left-hand navigation, pinned to the local (management) cluster.

> For production deployments, publish the extension bundle as a Helm chart and add it to Rancher's extension catalog. See the [Rancher Extension publishing guide](https://extensions.rancher.io/extensions/next/publishing) for details.

---

## Configuration reference

All values are set via `--set key=value` on `helm install/upgrade` or in a `values.yaml` file.

| Key | Default | Description |
|---|---|---|
| `image.repository` | `rancher-secrets-manager` | Container image repository |
| `image.tag` | *(chart appVersion)* | Container image tag |
| `image.pullPolicy` | `IfNotPresent` | Image pull policy |
| `rancher.url` | `https://rancher.cattle-system.svc` | Rancher API URL as seen from inside the cluster |
| `rancher.insecureTLS` | `false` | Skip TLS verification — for development only |
| `rancher.caBundle` | `""` | Path to a PEM CA bundle mounted in the pod |
| `leaderElection.enabled` | `true` | Enable leader election for HA deployments |
| `replicaCount` | `1` | Number of controller replicas |
| `resources.requests.cpu` | `50m` | CPU request |
| `resources.requests.memory` | `64Mi` | Memory request |
| `resources.limits.cpu` | `500m` | CPU limit |
| `resources.limits.memory` | `256Mi` | Memory limit |

---

## Usage

### Create a source secret

The source secret must exist in the management cluster before creating a `ManagedSecret`. Any namespace works, but `cattle-secrets-system` is a natural home:

```bash
kubectl create secret generic my-database-password \
  --namespace cattle-secrets-system \
  --from-literal=password='s3cr3t!'
```

### Create a ManagedSecret

```yaml
apiVersion: secrets.cattle.io/v1alpha1
kind: ManagedSecret
metadata:
  name: my-database-password
spec:
  secretRef:
    name: my-database-password        # source Secret name
    namespace: cattle-secrets-system  # source Secret namespace

  targets:
    # Push to a specific cluster by its Rancher display name
    - clusterName: production-eu
      namespace: app

    # Push to all clusters with a matching label, into the "app" namespace
    - clusterSelector:
        matchLabels:
          environment: staging
      namespace: app

    # Override the secret name on the downstream cluster
    - clusterName: production-us
      namespace: app
      secretName: db-password-override
```

Apply it:

```bash
kubectl apply -f managedsecret.yaml
```

### Check sync status

```bash
# Summary view (shows target and synced counts)
kubectl get managedsecret my-database-password
# NAME                    SOURCE SECRET          TARGETS   SYNCED   AGE
# my-database-password    my-database-password   3         3        30s

# Detailed per-cluster status
kubectl get managedsecret my-database-password \
  -o jsonpath='{.status.syncStatus}' | jq .
```

A `Synced` entry looks like:

```json
[
  {
    "clusterName": "production-eu",
    "clusterId": "c-m-abc12345",
    "namespace": "app",
    "secretName": "my-database-password",
    "status": "Synced",
    "lastSyncTime": "2024-01-15T10:05:00Z"
  }
]
```

If a target shows `Failed`, the `message` field contains the reason:

```bash
kubectl get managedsecret my-database-password \
  -o jsonpath='{.status.syncStatus[?(@.status=="Failed")]}' | jq .
```

### Update a secret

Update the source secret — the controller watches it and re-syncs all targets automatically:

```bash
kubectl create secret generic my-database-password \
  --namespace cattle-secrets-system \
  --from-literal=password='new-s3cr3t!' \
  --dry-run=client -o yaml | kubectl apply -f -
```

---

## Uninstall

```bash
# Remove the controller and RBAC
helm uninstall rancher-secrets-manager -n cattle-secrets-system

# Remove the CRD (this also deletes all ManagedSecret resources)
kubectl delete crd managedsecrets.secrets.cattle.io

# Remove the namespace
kubectl delete namespace cattle-secrets-system
```

> **Warning:** Deleting the CRD removes all `ManagedSecret` objects. Secrets already synced to downstream clusters are **not** deleted — they become unmanaged.

---

## Development

### Requirements

- Go 1.22+
- Docker + k3d
- Helm 3
- kubectl, Node 20, yarn

### Local dev environment

Spins up a management cluster with Rancher 2.14 and two downstream clusters, all on a shared Docker network:

```bash
./scripts/setup-dev.sh          # create everything (~5 min)
./scripts/setup-dev.sh teardown # destroy
```

After the script completes, follow its output to set `KUBECONFIG`, `RANCHER_URL`, and `RANCHER_TOKEN`.

### Build & run

```bash
make build                        # compile to bin/manager
make docker-build IMG=rsm:dev     # build container image

# Run directly against your current kubeconfig
RANCHER_URL=https://rancher.cattle-system.svc make run
```

### Generate CRD manifests

After changing types in `pkg/api/v1alpha1/`:

```bash
make generate    # regenerates zz_generated.deepcopy.go
make manifests   # regenerates charts/rancher-secrets-manager/crds/
```

### Tests

```bash
go test ./...                                         # unit tests
go test ./test/e2e/... -v -timeout 10m                # e2e (needs RANCHER_URL + RANCHER_TOKEN)
go test -run TestManagedSecret ./pkg/controller/...   # single test
```

### UI extension

```bash
cd ui
yarn install
API=https://your-rancher-instance yarn dev   # dev server with HMR
yarn build-pkg                               # production bundle → dist-pkg/
yarn lint                                    # ESLint
```

## Architecture

```
rancher-secrets-manager/
├── cmd/manager/                       # controller entrypoint
├── pkg/
│   ├── api/v1alpha1/                  # ManagedSecret CRD types
│   ├── controller/managedsecret/      # reconciler
│   └── rancher/                       # cluster resolution + proxy client
├── charts/rancher-secrets-manager/
│   ├── crds/                          # CRD YAML (installed before templates)
│   └── templates/                     # Deployment, RBAC, ServiceAccount
├── ui/
│   └── pkg/rancher-secrets-manager/   # Rancher UI extension
└── scripts/
    └── setup-dev.sh                   # local dev environment
```

See [CLAUDE.md](CLAUDE.md) for full architecture and design notes.
