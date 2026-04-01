BINARY_NAME ?= freeradius-operator
IMAGE_NAME  ?= freeradius-operator:dev
KIND_CLUSTER ?= freeradius-dev

# Go build settings
GOFLAGS ?=
GOARCH  ?= $(shell go env GOARCH)
GOOS    ?= $(shell go env GOOS)

GOLANGCI_LINT ?= $(shell which golangci-lint 2>/dev/null || echo "$(shell go env GOPATH)/bin/golangci-lint")

.PHONY: generate manifests build test lint dev-up dev-down load-image dev-run

## generate: Run controller-gen to generate DeepCopy methods and other code.
generate:
	controller-gen object:headerFile="hack/boilerplate.go.txt" paths="./..."

## manifests: Generate CRD and RBAC manifests.
manifests:
	controller-gen crd rbac:roleName=manager-role paths="./..." output:crd:artifacts:config=config/crd output:rbac:artifacts:config=config/rbac

## build: Compile the operator binary.
build:
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build $(GOFLAGS) -o bin/$(BINARY_NAME) ./cmd/operator/...

## test: Run all unit and property-based tests.
test:
	go test ./... -count=1

## lint: Run golangci-lint.
lint: $(GOLANGCI_LINT)
	$(GOLANGCI_LINT) run ./...

$(GOLANGCI_LINT):
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.8

## dev-up: Start the local kind-in-Docker-Compose development environment.
dev-up:
	docker compose up -d

## dev-down: Tear down the local development environment and remove volumes.
dev-down:
	docker compose down -v

## load-image: Build and load the operator image into the kind cluster.
load-image: build
	docker build -t $(IMAGE_NAME) .
	kind load docker-image $(IMAGE_NAME) --name $(KIND_CLUSTER)

## dev-run: Run the operator locally against the kind cluster.
dev-run:
	KUBECONFIG=./dev/kubeconfig go run ./cmd/operator/...
