---
title: Getting Started
nav_order: 2
---

# Getting Started
{: .no_toc }

Get FreeRADIUS running on your Kubernetes cluster in under five minutes.
{: .fs-6 .fw-300 }

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}

---

## Prerequisites

| Requirement | Version |
|:------------|:--------|
| Kubernetes cluster | 1.28+ |
| kubectl | 1.28+ |
| Go (for building from source) | 1.22+ |
| kind (for local development) | 0.20+ |

## Installation

### Install CRDs

```bash
kubectl apply -f config/crd/
```

Verify the CRDs are registered:

```bash
kubectl get crds | grep radius.operator.io
```

Expected output:

```
radiusclusters.radius.operator.io    2026-04-01T00:00:00Z
radiusclients.radius.operator.io     2026-04-01T00:00:00Z
radiuspolicies.radius.operator.io    2026-04-01T00:00:00Z
```

### Run the Operator

**Option A: Run locally against a kind cluster**

```bash
# Create a kind cluster with RADIUS ports exposed
kind create cluster \
  --name freeradius-dev \
  --config dev/kind-config.yaml \
  --kubeconfig dev/kubeconfig \
  --wait 120s

export KUBECONFIG=$(pwd)/dev/kubeconfig

# Build and load the operator image
make load-image

# Run the operator
make dev-run
```

**Option B: Build and deploy the container image**

```bash
# Build the operator binary
make build

# Build the container image
docker build -t freeradius-operator:latest .

# Deploy to your cluster (adapt as needed)
kubectl apply -f config/rbac/
kubectl create deployment freeradius-operator \
  --image=freeradius-operator:latest \
  --namespace=freeradius-system
```

## Your First RadiusCluster

### 1. Create a namespace

```bash
kubectl create namespace radius-demo
```

### 2. Create the shared secret

```bash
kubectl create secret generic my-shared-secret \
  --namespace=radius-demo \
  --from-literal=secret=testing123
```

### 3. Apply a minimal RadiusCluster

```yaml
# cluster.yaml
apiVersion: radius.operator.io/v1alpha1
kind: RadiusCluster
metadata:
  name: demo
  namespace: radius-demo
spec:
  image: freeradius/freeradius-server:3.2.3
  replicas: 1
```

```bash
kubectl apply -f cluster.yaml
```

### 4. Register a RADIUS client

```yaml
# client.yaml
apiVersion: radius.operator.io/v1alpha1
kind: RadiusClient
metadata:
  name: my-switch
  namespace: radius-demo
spec:
  clusterRef: demo
  ip: 192.168.1.0/24
  secretRef:
    name: my-shared-secret
    key: secret
  nasType: other
```

```bash
kubectl apply -f client.yaml
```

### 5. Verify the deployment

```bash
# Check the RadiusCluster status
kubectl get radiuscluster demo -n radius-demo -o wide

# Check the generated pods
kubectl get pods -n radius-demo -l app.kubernetes.io/name=demo

# Inspect the rendered ConfigMap
kubectl get configmap demo-config -n radius-demo -o yaml
```

You should see a running FreeRADIUS pod with the rendered configuration mounted.

### 6. Test authentication

If you're running on kind with the dev configuration, ports 30812 (auth) and 30813 (acct) are exposed on the host:

```bash
# Using radtest (install via freeradius-utils)
radtest testuser testpass localhost:30812 0 testing123
```

## What Happens Under the Hood

When you apply these resources, the operator:

1. Detects the new `RadiusCluster` and sets its status to `Progressing`
2. Lists all `RadiusClient` and `RadiusPolicy` resources with a matching `clusterRef`
3. Resolves all referenced Kubernetes Secrets
4. Invokes the **ConfigRenderer** to produce FreeRADIUS configuration files
5. Creates or updates a **ConfigMap** with the rendered configuration
6. Creates or updates a **Deployment** with the FreeRADIUS container, config volumes, and secret mounts
7. Creates or updates a **Service** exposing ports 1812/UDP and 1813/UDP
8. Optionally creates an **HPA** if autoscaling is enabled
9. Updates the `RadiusCluster` status to `Available`

Any subsequent change to any of the three CRDs triggers this reconciliation loop again, producing updated configuration and a rolling pod restart.

## Next Steps

- [Concepts](/freeradius-k8s-operator/concepts/) — RADIUS fundamentals for Kubernetes engineers (and vice versa)
- [Architecture](/freeradius-k8s-operator/architecture/) — Understand how the operator is structured
- [CRD Reference](/freeradius-k8s-operator/reference/radiuscluster/) — Full specification for all custom resources
- [Modules Guide](/freeradius-k8s-operator/guides/modules/) — Configure SQL, LDAP, EAP, REST, and Redis backends
- [Examples](/freeradius-k8s-operator/examples/) — Complete deployment scenarios (campus WiFi, ISP, VPN)
- [Development](/freeradius-k8s-operator/development/) — Build, test, and contribute to the project
