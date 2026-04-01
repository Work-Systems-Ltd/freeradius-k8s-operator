---
title: Home
layout: home
nav_order: 1
---

# FreeRADIUS Kubernetes Operator
{: .fs-9 }

Declarative RADIUS infrastructure on Kubernetes. Define clusters, clients, and policies as native custom resources — the operator handles the rest.
{: .fs-6 .fw-300 }

[Get Started](/freeradius-k8s-operator/getting-started/){: .btn .btn-primary .fs-5 .mb-4 .mb-md-0 .mr-2 }
[View on GitHub](https://github.com/Work-Systems-Ltd/freeradius-k8s-operator){: .btn .fs-5 .mb-4 .mb-md-0 }

---

## Why an Operator?

FreeRADIUS is the world's most widely deployed RADIUS server. Configuring it traditionally means managing dozens of flat files across machines — a process that doesn't translate well to cloud-native environments. This operator bridges that gap.

Instead of writing `radiusd.conf`, `clients.conf`, and `mods-enabled/*` by hand, you declare your intent through three Kubernetes custom resources. The operator reconciles those declarations into a fully configured, running FreeRADIUS deployment.

### What You Get

| Capability | How It Works |
|:-----------|:-------------|
| **Declarative Configuration** | Define your RADIUS infrastructure with `RadiusCluster`, `RadiusClient`, and `RadiusPolicy` CRDs. No shell access, no config file editing. |
| **Secure Secret Handling** | Shared secrets and credentials are referenced via Kubernetes Secrets and mounted as read-only volumes. Plaintext values never appear in ConfigMaps or CRD specs. |
| **Automatic Reconciliation** | Change a client IP or add a policy — the operator detects the change, re-renders configuration, and rolls out updated pods with zero manual steps. |
| **Horizontal Autoscaling** | Enable HPA-based autoscaling directly in the `RadiusCluster` spec. The operator manages the HorizontalPodAutoscaler lifecycle for you. |
| **Rolling Updates** | Deployment updates use `MaxUnavailable=0, MaxSurge=1` by default, ensuring at least one healthy pod serves traffic at all times. |
| **Observability** | Prometheus metrics (`freeradius_operator_reconcile_total`, `freeradius_operator_reconcile_duration_seconds`) are exposed on `:8080/metrics` out of the box. |

### Custom Resources at a Glance

```
RadiusCluster          RadiusClient            RadiusPolicy
┌────────────────┐     ┌────────────────┐      ┌────────────────┐
│ image           │     │ clusterRef     │      │ clusterRef     │
│ replicas        │◄────│ ip / CIDR      │      │ stage          │
│ modules[]       │     │ secretRef      │  ┌──►│ priority       │
│ tls             │◄────│ nasType        │  │   │ match{}        │
│ autoscaling     │     └────────────────┘  │   │ actions[]      │
│ resources       │                         │   └────────────────┘
│ probes          │◄────────────────────────┘
└────────────────┘
```

`RadiusClient` and `RadiusPolicy` resources reference a `RadiusCluster` via `clusterRef`. When any of the three resources change, the operator re-renders the full FreeRADIUS configuration and performs a rolling update.

---

## Quick Example

### 1. Define a cluster

```yaml
apiVersion: radius.operator.io/v1alpha1
kind: RadiusCluster
metadata:
  name: production
  namespace: radius
spec:
  image: freeradius/freeradius-server:3.2.3
  replicas: 3
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

### 2. Register a network device

```yaml
apiVersion: radius.operator.io/v1alpha1
kind: RadiusClient
metadata:
  name: core-switch
  namespace: radius
spec:
  clusterRef: production
  ip: 10.0.1.0/24
  secretRef:
    name: switch-secret
    key: shared-secret
  nasType: cisco
```

### 3. Add an authorization policy

```yaml
apiVersion: radius.operator.io/v1alpha1
kind: RadiusPolicy
metadata:
  name: reject-unknown-users
  namespace: radius
spec:
  clusterRef: production
  stage: authorize
  priority: 100
  match:
    none:
      - attribute: User-Name
        operator: "=~"
        value: "^[a-zA-Z0-9._-]+$"
  actions:
    - type: reject
```

### 4. Apply and go

```bash
kubectl apply -f config/crd/
kubectl apply -f examples/
```

The operator reconciles the resources, renders the FreeRADIUS configuration, and deploys pods that are ready to authenticate.

---

## New to RADIUS?

If you're a Kubernetes engineer who hasn't worked with RADIUS before, start with the [Concepts](/freeradius-k8s-operator/concepts/) page. It explains the AAA model, how RADIUS processing stages work, and maps every RADIUS concept to its Kubernetes equivalent.

## Project Status

This project is in **alpha** (`v1alpha1`). The API surface is stabilizing but may change between releases. It is suitable for development, testing, and non-critical deployments.

{: .warning }
> CRD schemas may change in future versions. Always review release notes before upgrading.

## Contributing

Contributions are welcome! See the [Development](/freeradius-k8s-operator/development/) page and [CONTRIBUTING.md](https://github.com/Work-Systems-Ltd/freeradius-k8s-operator/blob/master/CONTRIBUTING.md) for guidelines.
