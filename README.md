# freeradius-k8s-operator

[![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![Kubernetes](https://img.shields.io/badge/Kubernetes-1.28+-326CE5?logo=kubernetes&logoColor=white)](https://kubernetes.io)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)
[![Docs](https://img.shields.io/badge/docs-GitHub_Pages-blue?logo=github)](https://tbotnz.github.io/freeradius-k8s-operator/)

A Kubernetes operator for managing FreeRADIUS deployments, clients, and policies via custom resources.

Define your RADIUS infrastructure declaratively with three CRDs — `RadiusCluster`, `RadiusClient`, and `RadiusPolicy` — and let the operator handle configuration rendering, secret mounting, deployments, and rolling updates.

## Quick Start

### 1. Install CRDs

```bash
kubectl apply -f config/crd/
```

### 2. Create a RadiusCluster

```yaml
apiVersion: radius.operator.io/v1alpha1
kind: RadiusCluster
metadata:
  name: my-radius
  namespace: default
spec:
  image: freeradius/freeradius-server:3.2.3
  replicas: 2
  modules:
    - name: sql
      type: sql
      enabled: true
      sql:
        driver: postgresql
        server: db.internal
        port: 5432
        database: radius
        credentialsRef:
          name: db-credentials
          key: password
```

### 3. Register a RADIUS client (NAS device)

```yaml
apiVersion: radius.operator.io/v1alpha1
kind: RadiusClient
metadata:
  name: core-switch
  namespace: default
spec:
  clusterRef: my-radius
  ip: 10.0.1.0/24
  secretRef:
    name: switch-secret
    key: shared-secret
  nasType: cisco
```

### 4. Add an authorization policy

```yaml
apiVersion: radius.operator.io/v1alpha1
kind: RadiusPolicy
metadata:
  name: vlan-assignment
  namespace: default
spec:
  clusterRef: my-radius
  stage: post-auth
  priority: 100
  match:
    all:
      - attribute: User-Name
        operator: "=~"
        value: "^admin-"
  actions:
    - type: set
      attribute: Tunnel-Type
      value: VLAN
    - type: set
      attribute: Tunnel-Private-Group-Id
      value: "100"
```

### 5. Check status

```bash
# Overview of all RADIUS resources
kubectl get radiusclusters,radiusclients,radiuspolicies

# Detailed cluster status
kubectl get radiuscluster my-radius -o wide

# View rendered FreeRADIUS configuration
kubectl get configmap my-radius-config -o yaml

# Check pods
kubectl get pods -l app.kubernetes.io/name=my-radius
```

## Usage

### Manage clusters

```bash
# Create or update
kubectl apply -f cluster.yaml

# Scale replicas
kubectl patch radiuscluster my-radius --type merge -p '{"spec":{"replicas":3}}'

# Update FreeRADIUS image
kubectl patch radiuscluster my-radius --type merge \
  -p '{"spec":{"image":"freeradius/freeradius-server:3.2.4"}}'

# Enable autoscaling
kubectl patch radiuscluster my-radius --type merge \
  -p '{"spec":{"autoscaling":{"enabled":true,"minReplicas":2,"maxReplicas":10}}}'

# Delete (cascades to Deployment, Service, ConfigMap, HPA)
kubectl delete radiuscluster my-radius
```

### Manage clients

```bash
# List clients for a cluster
kubectl get radiusclients

# Add a client
kubectl apply -f client.yaml

# Remove a client (triggers config re-render)
kubectl delete radiusclient core-switch
```

### Manage policies

```bash
# List policies
kubectl get radiuspolicies

# Apply a policy
kubectl apply -f policy.yaml

# Check if a policy is valid
kubectl get radiuspolicy vlan-assignment -o jsonpath='{.status.conditions}'
```

### Secrets

Shared secrets and credentials are always stored in Kubernetes Secrets and mounted as read-only volumes — never embedded in ConfigMaps.

```bash
# Create a shared secret for a NAS device
kubectl create secret generic switch-secret \
  --from-literal=shared-secret='MyR@diusKey'

# Create database credentials for a SQL module
kubectl create secret generic db-credentials \
  --from-literal=password='dbP@ssw0rd'

# Create a TLS certificate for EAP
kubectl create secret tls eap-cert \
  --cert=server.crt --key=server.key
```

### Troubleshooting

```bash
# Check cluster conditions (Available, Progressing, Degraded)
kubectl get radiuscluster my-radius -o jsonpath='{.status.conditions}' | jq .

# Check for invalid clients or policies
kubectl get radiusclients -o custom-columns=NAME:.metadata.name,READY:.status.conditions[0].status
kubectl get radiuspolicies -o custom-columns=NAME:.metadata.name,READY:.status.conditions[0].status

# View operator logs
kubectl logs deploy/freeradius-operator -f

# View FreeRADIUS pod logs
kubectl logs -l app.kubernetes.io/name=my-radius

# Inspect rendered config
kubectl get configmap my-radius-config -o jsonpath='{.data.clients\.conf}'

# Check operator metrics
kubectl port-forward deploy/freeradius-operator 8080:8080
curl localhost:8080/metrics | grep freeradius_operator
```

## Documentation

Full documentation is available at **[tbotnz.github.io/freeradius-k8s-operator](https://tbotnz.github.io/freeradius-k8s-operator/)**, including:

- [Getting Started](https://tbotnz.github.io/freeradius-k8s-operator/getting-started/) — install and deploy your first cluster
- [Concepts](https://tbotnz.github.io/freeradius-k8s-operator/concepts/) — RADIUS fundamentals for Kubernetes engineers
- [Architecture](https://tbotnz.github.io/freeradius-k8s-operator/architecture/) — how the operator works under the hood
- [CRD Reference](https://tbotnz.github.io/freeradius-k8s-operator/reference/radiuscluster/) — full spec for all three custom resources
- [Guides](https://tbotnz.github.io/freeradius-k8s-operator/guides/modules/) — modules, autoscaling, TLS, security, observability, upgrading
- [Examples](https://tbotnz.github.io/freeradius-k8s-operator/examples/) — complete deployment scenarios (campus WiFi, ISP, VPN)
- [Troubleshooting](https://tbotnz.github.io/freeradius-k8s-operator/troubleshooting/) — common issues and fixes
- [FAQ](https://tbotnz.github.io/freeradius-k8s-operator/faq/)

## Development

### Prerequisites

- Go 1.22+
- Docker
- [kind](https://kind.sigs.k8s.io/) (`go install sigs.k8s.io/kind@latest` or grab a binary)
- kubectl

### Local cluster

```bash
# Create a kind cluster with RADIUS ports exposed
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

# Lint
make lint

# Build the operator binary
make build

# Build image and load into kind (no registry needed)
make load-image

# Run the operator locally against the kind cluster
make dev-run
```

### Code generation

```bash
# After modifying CRD types in api/v1alpha1/
make generate    # DeepCopy methods
make manifests   # CRD + RBAC YAML
```

### Tear down

```bash
kind delete cluster --name freeradius-dev
```

## Contributing

Contributions are welcome! See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines on development workflow, code standards, and how to submit pull requests.

## License

This project is licensed under the [Apache License 2.0](LICENSE).
