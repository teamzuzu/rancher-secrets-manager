# rancher-secrets-manager

Define a secret once in your Rancher management cluster. The controller automatically keeps it in sync across every downstream cluster and namespace you target — no per-cluster credentials, no manual copying.

---

## Installation

### What you'll install

| Component | What it does |
|---|---|
| **Controller** | Kubernetes operator that syncs secrets. Installed via Rancher Apps. |
| **UI Extension** | Adds a *Secrets Manager* page to the Rancher dashboard. Installed via Rancher Extensions. |

Both are served from a single Helm repository. Add it once; install each component separately.

---

### Step 1 — Add the Helm repository

Open the Rancher dashboard and navigate to **☰ → Apps → Repositories**.

Click **Create** and fill in:

| Field | Value |
|---|---|
| Name | `rancher-secrets-manager` |
| Index URL | `https://teamzuzu.github.io/rancher-secrets-manager` |

Click **Create**.

> **Tip:** This same repository also powers the Extensions catalog in the next step — you only need to add it once.

---

### Step 2 — Install the controller

1. Navigate to **☰ → Apps → Charts**.
2. Search for **Secrets Manager** and click the chart.
3. Click **Install**.
4. On the *Namespace* screen, set the namespace to `cattle-secrets-system` and tick *Create namespace* if it doesn't exist.
5. Leave all other values at their defaults and click **Install**.

Wait for the `rancher-secrets-manager` workload to show **Active** in the `cattle-secrets-system` namespace.

---

### Step 3 — Install the UI extension

1. Navigate to **☰ → Extensions**.
2. Click **⋮ → Manage Repositories** and confirm `rancher-secrets-manager` is listed.  
   *(It was added in Step 1 — the same repository serves both the app chart and the extension.)*
3. Go back to **Extensions → Available**.
4. Find **Secrets Manager** and click **Install**.

After a moment, **Secrets Manager** appears in the left-hand navigation under the local cluster.

---

### Step 4 — Create your first managed secret

Before creating a `ManagedSecret`, a source `Secret` must exist in the management cluster:

```bash
kubectl create secret generic my-api-key \
  --namespace cattle-secrets-system \
  --from-literal=token='abc123'
```

Then open **Secrets Manager** in the Rancher dashboard and click **Create**.  
Or apply YAML directly:

```yaml
apiVersion: secrets.cattle.io/v1alpha1
kind: ManagedSecret
metadata:
  name: my-api-key
spec:
  secretRef:
    name: my-api-key
    namespace: cattle-secrets-system
  targets:
    - clusterName: production-eu   # Rancher display name
      namespace: app
    - clusterSelector:
        matchLabels:
          environment: staging
      namespace: app
```

The controller syncs within seconds. Open the resource in the UI or run:

```bash
kubectl get managedsecret my-api-key
# NAME          SOURCE SECRET   TARGETS   SYNCED   AGE
# my-api-key    my-api-key      2         2        8s
```

---

## Configuration

All settings are available in the Rancher Apps install form. The most commonly changed values:

| Setting | Default | When to change |
|---|---|---|
| `rancher.url` | `https://rancher.cattle-system.svc` | Only if Rancher runs in a non-standard namespace or with a custom hostname |
| `rancher.insecureTLS` | `false` | Local dev environments with self-signed certs |
| `rancher.caBundle` | *(none)* | Rancher uses a private CA — provide the path to the PEM file mounted in the pod |
| `replicaCount` | `1` | Set to `2` for high-availability deployments |

---

## Targeting clusters

Each target in `spec.targets` specifies *which* clusters receive the secret and *where* to put it:

```yaml
targets:
  # By Rancher display name (exact match)
  - clusterName: production-eu
    namespace: app

  # By label selector (all matching clusters)
  - clusterSelector:
      matchLabels:
        environment: staging
        region: eu
    namespace: app

  # Override the secret name on the downstream cluster
  - clusterName: production-us
    namespace: payments
    secretName: stripe-key
```

`clusterName` and `clusterSelector` are mutually exclusive within a single target entry.

---

## Checking sync status

**In the UI:** open the ManagedSecret detail page — the *Sync Status* tab shows each cluster, its current state (Synced / Failed / Pending), and the timestamp of the last successful sync.

**From kubectl:**

```bash
# Summary
kubectl get managedsecret my-api-key

# Full per-cluster detail
kubectl get managedsecret my-api-key -o jsonpath='{.status.syncStatus}' | jq .
```

When a target shows `Failed`, the `message` field explains why:

```bash
kubectl get managedsecret my-api-key \
  -o jsonpath='{.status.syncStatus[?(@.status=="Failed")].message}'
```

---

## Uninstall

1. **Remove the controller** — in Rancher, go to **☰ → Apps → Installed Apps**, find `rancher-secrets-manager`, and click **Delete**.
2. **Remove the UI extension** — go to **☰ → Apps → Installed Apps** (on the **local** cluster), find `rancher-secrets-manager-ui`, and click **Delete**.

   > **Note:** The *Uninstall* button in the Extensions tab does not work for community extensions in Rancher 2.14. Use Apps → Installed Apps instead.

3. **Remove the CRD** (optional — this deletes all ManagedSecret objects):

```bash
kubectl delete crd managedsecrets.secrets.cattle.io
```

> Secrets already synced to downstream clusters are **not** deleted when you remove the controller or CRD — they simply become unmanaged.

---

## How it works

```
ManagedSecret → controller → Rancher API proxy → cattle-cluster-agent → downstream Secret
```

1. The controller reads a `ManagedSecret` and resolves its target list by querying `management.cattle.io/v3 Cluster` objects.
2. For each `(cluster, namespace)` pair it calls Rancher's built-in proxy endpoint (`/k8s/clusters/<id>/`) using its own ServiceAccount token — no extra credentials on downstream clusters.
3. Sync state is written back to `.status.syncStatus` after every reconcile.

The controller also watches the referenced source `Secret` and re-queues any `ManagedSecret` that uses it whenever its data changes.

---

## Releasing a new version

Tag the commit and push — the release workflow does the rest:

```bash
git tag v0.2.0
git push origin v0.2.0
```

The workflow will:
- Build and push the container image to `ghcr.io/teamzuzu/rancher-secrets-manager`
- Package and publish both Helm charts to GitHub Pages
- Build and publish the UI extension bundle

---

## Development

See [CLAUDE.md](CLAUDE.md) for architecture details.

### Requirements

Go 1.22+, Docker + k3d, Helm 3, kubectl, Node 20, yarn

### Local dev environment

```bash
./scripts/setup-dev.sh          # management cluster + 2 downstream clusters (~5 min)
./scripts/setup-dev.sh teardown
```

### Build & test

```bash
make build                             # compile controller
make docker-build IMG=rsm:dev          # build image
go test ./...                          # unit tests
RANCHER_URL=https://... make run       # run locally against current kubeconfig
```

### UI extension

```bash
cd ui
yarn install
API=https://your-rancher yarn dev      # dev server with hot reload
yarn build-pkg                         # production bundle → dist-pkg/
```
