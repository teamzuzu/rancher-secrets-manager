# rancher-secrets-manager

A Rancher plugin that distributes secrets from the management cluster to downstream clusters. Define a `ManagedSecret` once; the controller keeps the secret in sync across every targeted cluster and namespace.

## How it works

1. A `ManagedSecret` (cluster-scoped CRD) references a source `Secret` in the management cluster and a list of targets — each target specifies a downstream cluster (by name or label selector) and a namespace.
2. The controller resolves targets by listing `management.cattle.io/v3 Cluster` objects.
3. For each `(cluster, namespace)` pair it creates or updates the secret by proxying through Rancher's built-in API endpoint (`/k8s/clusters/<cluster-id>/`), using the controller's own service account token — no extra credentials needed on downstream clusters.
4. Sync state is recorded per-cluster in `.status.syncStatus`.

```
ManagedSecret → controller → Rancher API proxy → cattle-cluster-agent → downstream Secret
```

## Installation

```bash
helm repo add rancher-secrets-manager https://teamzuzu.github.io/rancher-secrets-manager
helm install rancher-secrets-manager rancher-secrets-manager/rancher-secrets-manager \
  -n cattle-secrets-system --create-namespace
```

Or from source:

```bash
helm install rancher-secrets-manager charts/rancher-secrets-manager \
  -n cattle-secrets-system --create-namespace \
  --set rancher.url=https://rancher.cattle-system.svc
```

## Usage

```yaml
apiVersion: secrets.cattle.io/v1alpha1
kind: ManagedSecret
metadata:
  name: my-db-password
spec:
  secretRef:
    name: my-db-password        # source Secret in the management cluster
    namespace: cattle-secrets-system
  targets:
    # Push to a specific cluster by display name, into the "app" namespace
    - clusterName: production-eu
      namespace: app

    # Push to all clusters tagged environment=staging, into the "app" namespace
    - clusterSelector:
        matchLabels:
          environment: staging
      namespace: app

    # Override the secret name in the downstream cluster
    - clusterName: production-us
      namespace: app
      secretName: db-password-override
```

Check sync status:

```bash
kubectl get managedsecret my-db-password
# NAME              SOURCE SECRET    TARGETS   SYNCED   AGE
# my-db-password    my-db-password   3         3        2m

kubectl get managedsecret my-db-password -o jsonpath='{.status.syncStatus}' | jq .
```

## Development

### Requirements

- Go 1.22+
- Docker + k3d
- Helm 3
- kubectl

### Local dev environment

Spins up a management cluster with Rancher 2.14 and two downstream clusters, all connected via a shared Docker network:

```bash
./scripts/setup-dev.sh          # create everything (~5 min)
./scripts/setup-dev.sh teardown # destroy
```

Once up, source the output to get `RANCHER_URL` and `RANCHER_TOKEN` env vars.

### Build & run

```bash
make build                        # compile to bin/manager
make docker-build IMG=rsm:dev     # build container image

# Run against your current kubeconfig (requires RANCHER_URL)
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
go test ./...                                        # unit tests
go test ./test/e2e/... -v -timeout 10m               # e2e (needs RANCHER_URL + RANCHER_TOKEN)
go test -run TestManagedSecret ./pkg/controller/...  # single test
```

## Architecture

```
rancher-secrets-manager/
├── cmd/manager/              # controller entrypoint
├── pkg/
│   ├── api/v1alpha1/         # ManagedSecret CRD types
│   ├── controller/managedsecret/  # reconciler
│   └── rancher/              # cluster resolution + proxy client
├── charts/rancher-secrets-manager/
│   ├── crds/                 # CRD YAML
│   └── templates/            # Deployment, RBAC, ServiceAccount
└── scripts/
    └── setup-dev.sh          # local dev environment
```

See [CLAUDE.md](CLAUDE.md) for full architecture notes.
