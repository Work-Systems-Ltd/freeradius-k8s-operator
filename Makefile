BINARY_NAME  ?= freeradius-operator
IMAGE_NAME   ?= freeradius-operator:dev
KIND_CLUSTER ?= freeradius-dev

GOFLAGS ?=
GOARCH  ?= $(shell go env GOARCH)
GOOS    ?= $(shell go env GOOS)
GOPATH_BIN ?= $(shell go env GOPATH)/bin

GOLANGCI_LINT_VERSION ?= v1.64.8
GOLANGCI_LINT         ?= $(GOPATH_BIN)/golangci-lint

.PHONY: all generate manifests build test test-e2e fmt lint setup-hooks dev-up dev-down load-image dev-run

all: fmt lint test build

generate:
	controller-gen object paths="./..."

manifests:
	controller-gen crd rbac:roleName=manager-role paths="./..." \
		output:crd:artifacts:config=config/crd \
		output:rbac:artifacts:config=config/rbac

build:
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build $(GOFLAGS) -o bin/$(BINARY_NAME) ./cmd/operator/...

test:
	go test $(shell go list ./... | grep -v /e2e) -count=1

test-e2e:
	go test ./e2e/... -count=1

fmt:
	gofmt -w $(shell find . -name '*.go' -not -path './vendor/*' -not -path '*/zz_generated*')
	$(GOPATH_BIN)/goimports -w -local github.com/example/freeradius-operator \
		$(shell find . -name '*.go' -not -path './vendor/*' -not -path '*/zz_generated*')

lint: $(GOLANGCI_LINT)
	$(GOLANGCI_LINT) run ./...

$(GOLANGCI_LINT):
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

setup-hooks:
	git config core.hooksPath .githooks

dev-up:
	docker compose up -d

dev-down:
	docker compose down -v

load-image: build
	docker build -t $(IMAGE_NAME) .
	kind load docker-image $(IMAGE_NAME) --name $(KIND_CLUSTER)

dev-run:
	KUBECONFIG=./dev/kubeconfig OPERATOR_IMAGE=$(IMAGE_NAME) go run ./cmd/operator/...
