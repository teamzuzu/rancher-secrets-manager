IMG ?= rancher-secrets-manager:latest
CONTROLLER_GEN_VERSION ?= v0.16.4
ENVTEST_VERSION ?= release-0.19

.PHONY: all build test lint generate manifests docker-build run help

all: build

## Build the controller binary.
build:
	go build -o bin/manager ./cmd/manager

## Run unit tests.
test:
	go test ./... -v

## Run controller against the current kubeconfig context.
run:
	go run ./cmd/manager \
		--rancher-url=$(RANCHER_URL) \
		--insecure-tls=$(INSECURE_TLS)

## Lint with golangci-lint.
lint:
	golangci-lint run ./...

## Generate deep copy code (requires controller-gen).
generate: controller-gen
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

## Generate CRD and RBAC manifests (requires controller-gen).
manifests: controller-gen
	$(CONTROLLER_GEN) crd rbac:roleName=rancher-secrets-manager-role webhook paths="./..." \
		output:crd:artifacts:config=charts/rancher-secrets-manager/crds \
		output:rbac:artifacts:config=config/rbac

## Build the Docker image.
docker-build:
	docker build -t $(IMG) .

## Push the Docker image.
docker-push:
	docker push $(IMG)

## Install CRDs into the current cluster.
install: manifests
	kubectl apply -f charts/rancher-secrets-manager/crds/

## Uninstall CRDs from the current cluster.
uninstall:
	kubectl delete -f charts/rancher-secrets-manager/crds/ --ignore-not-found

CONTROLLER_GEN = $(shell pwd)/bin/controller-gen

controller-gen:
	@test -s $(CONTROLLER_GEN) || \
		GOBIN=$(shell pwd)/bin go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_GEN_VERSION)

help:
	@grep -E '^## ' Makefile | sed 's/## //'
