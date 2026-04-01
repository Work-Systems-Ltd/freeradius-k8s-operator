# RadiusCluster

The primary resource that defines a FreeRADIUS deployment.

---

## Overview

A `RadiusCluster` represents a logical FreeRADIUS deployment. The operator translates it into a Deployment, Service, ConfigMap, and optionally an HPA.

**API Group**: `radius.operator.io/v1alpha1`
**Kind**: `RadiusCluster`
**Short Name**: `rc`
**Scope**: Namespaced

## Full Example

```yaml
apiVersion: radius.operator.io/v1alpha1
kind: RadiusCluster
metadata:
  name: production
  namespace: radius
spec:
  image: freeradius/freeradius-server:3.2.3
  replicas: 3
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
  tls:
    enabled: true
    secretRef:
      name: radius-tls
      key: tls.crt
  probes:
    liveness:
      initialDelaySeconds: 10
      periodSeconds: 30
    readiness:
      initialDelaySeconds: 5
      periodSeconds: 10
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
    - name: ldap
      type: ldap
      enabled: true
      ldap:
        server: ldap://ldap.internal
        port: 389
        baseDN: "dc=example,dc=com"
        identity: "cn=admin,dc=example,dc=com"
        credentialsRef:
          name: ldap-credentials
          key: password
```

## Spec Fields

### `image` (required)

The container image for FreeRADIUS.

```yaml
image: freeradius/freeradius-server:3.2.3
```

| Type | Default |
|:-----|:--------|
| string | — |

### `replicas`

Number of FreeRADIUS pod replicas. Ignored when autoscaling is enabled.

```yaml
replicas: 3
```

| Type | Default | Minimum |
|:-----|:--------|:--------|
| int32 | 1 | 1 |

### `resources`

Standard Kubernetes resource requests and limits for the FreeRADIUS container.

```yaml
resources:
  requests:
    cpu: 250m
    memory: 256Mi
  limits:
    cpu: "1"
    memory: 512Mi
```

### `autoscaling`

Enables Horizontal Pod Autoscaler management.

| Field | Type | Default | Description |
|:------|:-----|:--------|:------------|
| `enabled` | bool | false | Whether to create an HPA |
| `minReplicas` | int32 | 1 | Minimum replica count |
| `maxReplicas` | int32 | 10 | Maximum replica count |
| `targetCPUUtilizationPercentage` | int32 | 80 | CPU target for scaling |

When `autoscaling.enabled` is `true`, the operator creates and manages an HPA. When set to `false`, any existing HPA is deleted and `spec.replicas` governs the replica count.

### `tls`

TLS configuration for RADIUS over TLS (RadSec).

| Field | Type | Description |
|:------|:-----|:------------|
| `enabled` | bool | Enable TLS |
| `secretRef` | [SecretRef](#secretref) | Reference to TLS certificate Secret |

### `probes`

Override default liveness and readiness probes.

```yaml
probes:
  liveness:
    initialDelaySeconds: 10
    periodSeconds: 30
  readiness:
    initialDelaySeconds: 5
    periodSeconds: 10
```

If not specified, the operator uses sensible defaults:
- **Readiness**: exec-based status check on port 1812
- **Liveness**: exec-based `radiusd -C` syntax validation

### `modules`

List of FreeRADIUS modules to enable. See the [Modules Guide](guides/modules/) for detailed configuration.

```yaml
modules:
  - name: sql
    type: sql
    enabled: true
    sql: { ... }
```

| Field | Type | Description |
|:------|:-----|:------------|
| `name` | string | Unique module name |
| `type` | enum | One of: `sql`, `ldap`, `eap`, `rest`, `redis` |
| `enabled` | bool | Whether the module is active |
| `sql` | SQLConfig | SQL-specific settings (when type=sql) |
| `ldap` | LDAPConfig | LDAP-specific settings (when type=ldap) |
| `eap` | EAPConfig | EAP-specific settings (when type=eap) |
| `rest` | RESTConfig | REST-specific settings (when type=rest) |
| `redis` | RedisConfig | Redis-specific settings (when type=redis) |

## Status Fields

| Field | Type | Description |
|:------|:-----|:------------|
| `readyReplicas` | int32 | Number of pods in Ready state |
| `currentImage` | string | Container image of the active deployment |
| `podRestarts` | int32 | Total restart count across all pods |
| `conditions` | []Condition | Status conditions (see below) |

### Conditions

| Type | Description |
|:-----|:------------|
| `Available` | All resources reconciled; pods serving traffic |
| `Progressing` | Reconciliation in progress |
| `Degraded` | Missing secret or recoverable error; will retry |

## Common Types

### SecretRef

A reference to a key within a Kubernetes Secret.

```yaml
secretRef:
  name: my-secret    # Secret name in the same namespace
  key: password       # Key within the Secret's data
```

| Field | Type | Description |
|:------|:-----|:------------|
| `name` | string | Name of the Kubernetes Secret |
| `key` | string | Key within the Secret |

## Generated Resources

For a `RadiusCluster` named `production`, the operator creates:

| Resource | Name | Description |
|:---------|:-----|:------------|
| ConfigMap | `production-config` | Rendered FreeRADIUS configuration files |
| Deployment | `production` | FreeRADIUS pods |
| Service | `production` | ClusterIP service on UDP 1812 + 1813 |
| HPA | `production` | Only if `autoscaling.enabled: true` |
