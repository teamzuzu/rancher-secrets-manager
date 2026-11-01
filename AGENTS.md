# Repository Guidelines

## Project Structure & Module Organization

This repository contains a Go Kubernetes controller and a Rancher dashboard extension. `cmd/manager/` is the controller entry point; `pkg/api/v1alpha1/` defines `ManagedSecret`, `pkg/controller/managedsecret/` reconciles secrets, and `pkg/rancher/` handles downstream cluster access. The Vue/TypeScript extension lives in `ui/pkg/rancher-secrets-manager/`, including views, models, routing, and translations. Helm charts live under `charts/`; development tooling is in `scripts/`, and automation is in `.github/workflows/`.

## Build, Test, and Development Commands

Use Go 1.22+ and Node 20+ with Yarn.

- `make build`: compile the controller to `bin/manager`.
- `make test`: run Go tests with verbose output.
- `make lint`: run the configured golangci-lint checks.
- `make run RANCHER_URL=https://... INSECURE_TLS=false`: run against the current kubeconfig context.
- `make docker-build IMG=rancher-secrets-manager:dev`: build a container image.
- `helm lint charts/rancher-secrets-manager`: validate the controller chart.
- In `ui/`, run `yarn install --frozen-lockfile`, then `yarn dev`, `yarn lint`, or `yarn build-pkg` to develop, lint, or package the extension.
- `./scripts/setup-dev.sh`: provision a local Rancher environment with management and downstream clusters; requires Docker and cluster tooling.

## Coding Style & Naming Conventions

Format Go with `gofmt` and follow idiomatic exported `PascalCase` and unexported `camelCase` naming. Keep packages lowercase. Follow existing UI files: two-space indentation, single quotes, and semicolons; run ESLint through `yarn lint`. Keep resource view filenames aligned with the API type, such as `secrets.cattle.io.managedsecret.vue`.

## Testing Guidelines

Add Go tests beside implementation files as `*_test.go`, using `TestXxx` functions and the standard `testing` package. Run `go test ./... -race -coverprofile=coverage.out` to match CI. Coverage is collected without a configured minimum. No test files are currently checked in; the e2e workflow references `test/e2e/`, which is absent. Validate UI changes with linting, package builds, and manual Rancher checks.

## Commit & Contribution Guidelines

History uses Conventional Commit prefixes such as `fix(ui):`, `feat(ui):`, and `chore(release):`. Describe the concrete change in the subject. Follow `CLAUDE.md`: commit with the configured Git identity and push directly to `main`; do not create feature branches or pull requests. Record relevant validation and behavior changes in commit descriptions.

## Security & Configuration

Never commit secret values, Rancher tokens, or kubeconfigs. Check the active Kubernetes context before running cluster commands. Keep the controller CRD in `charts/rancher-secrets-manager/templates/crd.yaml` so Helm upgrades apply schema changes.
