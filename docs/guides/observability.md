---
title: Observability
parent: Guides
nav_order: 4
---

# Observability
{: .no_toc }

Monitor the operator and your RADIUS infrastructure with Prometheus metrics.
{: .fs-6 .fw-300 }

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}

---

## Operator Metrics

The operator exposes Prometheus metrics on `:8080/metrics`. These cover the operator's own reconciliation performance — not FreeRADIUS traffic metrics.

### Available Metrics

| Metric | Type | Labels | Description |
|:-------|:-----|:-------|:------------|
| `freeradius_operator_reconcile_total` | Counter | `namespace`, `name`, `kind`, `result` | Total reconciliation attempts. `result` is `success` or `error`. |
| `freeradius_operator_reconcile_duration_seconds` | Histogram | — | Time spent in each reconciliation loop. |

### Scrape Configuration

If you're using the Prometheus Operator, create a `ServiceMonitor`:

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: freeradius-operator
  namespace: freeradius-system
spec:
  selector:
    matchLabels:
      app: freeradius-operator
  endpoints:
    - port: metrics
      interval: 30s
      path: /metrics
```

For plain Prometheus, add a scrape config:

```yaml
scrape_configs:
  - job_name: freeradius-operator
    kubernetes_sd_configs:
      - role: pod
    relabel_configs:
      - source_labels: [__meta_kubernetes_pod_label_app]
        regex: freeradius-operator
        action: keep
      - source_labels: [__meta_kubernetes_pod_container_port_number]
        regex: "8080"
        action: keep
```

## Status Conditions

Beyond metrics, the operator writes structured conditions to each resource's status. These are queryable with `kubectl` and useful for alerting.

### Check cluster health

```bash
# Quick overview
kubectl get radiuscluster -n radius

# Detailed conditions
kubectl get radiuscluster production -n radius -o jsonpath='{.status.conditions}' | jq .
```

### Alert on degraded clusters

A `Degraded` condition means the operator detected a problem (usually a missing Secret) and is retrying. You can alert on this with a Prometheus rule that watches the `kube_customresource_status_condition` metric (if using kube-state-metrics with CRD support) or by polling the Kubernetes API.

## Useful kubectl Commands

```bash
# List all RADIUS resources
kubectl get rc,rcl,rp -n radius

# Watch reconciliation in real time
kubectl get radiuscluster -n radius -w

# Check pod health
kubectl get pods -n radius -l app.kubernetes.io/managed-by=freeradius-operator

# View operator logs
kubectl logs -n freeradius-system deploy/freeradius-operator -f

# Check pod restart count from status
kubectl get radiuscluster production -n radius \
  -o jsonpath='{.status.podRestarts}'
```
