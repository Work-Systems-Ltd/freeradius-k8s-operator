# freeradius-k8s-operator

A Kubernetes operator for managing FreeRADIUS deployments, clients, and policies via custom resources.

## Development

### Prerequisites

- Go 1.22+
- Docker
- [kind](https://kind.sigs.k8s.io/) (`go install sigs.k8s.io/kind@latest` or grab a binary)
- kubectl

### Local cluster

```bash
# Create a kind cluster
kind create cluster --name freeradius-dev --config dev/kind-config.yaml --kubeconfig dev/kubeconfig --wait 120s

# Point kubectl at it
export KUBECONFIG=$(pwd)/dev/kubeconfig

# Install CRDs
kubectl apply -f config/crd/
```

### Build and run

```bash
# Run tests
make test

# Build the operator binary
make build

# Build image and load into kind
make load-image

# Run the operator locally against the kind cluster
make dev-run
```

### Tear down

```bash
kind delete cluster --name freeradius-dev
```
