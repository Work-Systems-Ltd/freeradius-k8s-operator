---
title: Autoscaling
parent: Guides
nav_order: 2
---

# Autoscaling
{: .no_toc }

Scale FreeRADIUS pods automatically based on CPU utilization.
{: .fs-6 .fw-300 }

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}

---

## Overview

The operator manages a Kubernetes `HorizontalPodAutoscaler` (HPA) for each `RadiusCluster` that has autoscaling enabled. This lets your RADIUS infrastructure scale with demand — handling peak authentication loads without over-provisioning during quiet periods.

## Prerequisites

Your cluster must have the [Metrics Server](https://github.com/kubernetes-sigs/metrics-server) installed. Verify with:

```bash
kubectl top nodes
```

If this returns metrics, you're ready. If not, install the Metrics Server first.

## Enable Autoscaling

```yaml
apiVersion: radius.operator.io/v1alpha1
kind: RadiusCluster
metadata:
  name: production
  namespace: radius
spec:
  image: freeradius/freeradius-server:3.2.3
  replicas: 2
  resources:
    requests:
      cpu: 250m
      memory: 256Mi
    limits:
      cpu: "1"
      memory: 512Mi
  autoscaling:
    enabled: true
    minReplicas: 2
    maxReplicas: 10
    targetCPUUtilizationPercentage: 70
```

{: .note }
> When autoscaling is enabled, the `spec.replicas` field is ignored. The HPA controls the replica count between `minReplicas` and `maxReplicas`.

## Configuration

| Field | Default | Description |
|:------|:--------|:------------|
| `enabled` | `false` | Set to `true` to create the HPA |
| `minReplicas` | `1` | Floor for pod count — never scale below this |
| `maxReplicas` | `10` | Ceiling for pod count — never scale above this |
| `targetCPUUtilizationPercentage` | `80` | Target average CPU usage across all pods |

## How It Works

1. When `autoscaling.enabled` is `true`, the operator creates an HPA targeting the FreeRADIUS Deployment
2. The HPA monitors average CPU utilization across pods
3. When utilization exceeds the target, the HPA increases replicas (up to `maxReplicas`)
4. When utilization drops, the HPA decreases replicas (down to `minReplicas`)
5. If you set `autoscaling.enabled` back to `false`, the operator **deletes the HPA** and `spec.replicas` takes effect again

## Resource Requests Are Required

The HPA calculates CPU utilization as a percentage of the CPU **request**. If you don't set `resources.requests.cpu`, the HPA has no baseline and cannot make scaling decisions.

```yaml
resources:
  requests:
    cpu: 250m       # Required for HPA to work
    memory: 256Mi
```

## Sizing Guidelines

| Deployment Size | minReplicas | maxReplicas | Target CPU |
|:----------------|:------------|:------------|:-----------|
| Development | 1 | 3 | 80% |
| Small office (< 500 users) | 2 | 5 | 75% |
| Campus (500–5000 users) | 3 | 10 | 70% |
| Enterprise (5000+ users) | 5 | 20 | 65% |

These are starting points — adjust based on your observed authentication patterns and response time requirements.

## Disable Autoscaling

Set `enabled: false` and specify your desired replica count:

```yaml
spec:
  replicas: 3
  autoscaling:
    enabled: false
```

The operator deletes the HPA and sets the Deployment to 3 replicas.
