---
title: RadiusClient
parent: CRD Reference
nav_order: 2
---

# RadiusClient
{: .no_toc }

Defines a network device authorized to send RADIUS requests.
{: .fs-6 .fw-300 }

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}

---

## Overview

A `RadiusClient` maps to a `client` block in FreeRADIUS's `clients.conf`. Each RadiusClient declares a network device (switch, access point, VPN concentrator) that is permitted to authenticate against the RADIUS server.

**API Group**: `radius.operator.io/v1alpha1`
**Kind**: `RadiusClient`
**Short Name**: `rcl`
**Scope**: Namespaced

## Full Example

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
  metadata:
    location: building-a-floor-3
    role: distribution
```

## Spec Fields

### `clusterRef` (required)

Name of the `RadiusCluster` this client belongs to. Must be in the same namespace.

```yaml
clusterRef: production
```

| Type | Default |
|:-----|:--------|
| string | — |

### `ip` (required)

IPv4 address, IPv6 address, or CIDR range of the network device.

```yaml
# Single host
ip: 10.0.1.1

# CIDR range
ip: 10.0.1.0/24

# IPv6
ip: "2001:db8::1"
```

| Type | Default | Validation |
|:-----|:--------|:-----------|
| string | — | Must match IPv4, IPv6, or CIDR format |

### `secretRef` (required)

Reference to the Kubernetes Secret containing the shared secret for RADIUS authentication between the NAS and the server.

```yaml
secretRef:
  name: switch-secret
  key: shared-secret
```

| Field | Type | Description |
|:------|:-----|:------------|
| `name` | string | Name of the Kubernetes Secret |
| `key` | string | Key within the Secret's data |

{: .warning }
> The shared secret value is **never** written into the ConfigMap. It is mounted as a read-only file and referenced via `${file:...}` in the rendered configuration.

### `nasType`

The type of network access server. This is an informational field used by FreeRADIUS for vendor-specific behavior.

```yaml
nasType: cisco
```

| Type | Default |
|:-----|:--------|
| string | — |

Common values: `cisco`, `juniper`, `other`, `aruba`, `mikrotik`

### `metadata`

Arbitrary key-value labels for organizational purposes. These are rendered as comments in the generated `clients.conf` and are useful for tracking device location, role, or owner.

```yaml
metadata:
  location: dc-west-rack-42
  owner: network-team
  ticket: NET-1234
```

| Type | Default |
|:-----|:--------|
| map[string]string | — |

## Status

### Conditions

| Type | Description |
|:-----|:------------|
| `Ready` | Client is valid and the referenced cluster exists |
| `Invalid` | Validation failed — check the condition message for details |

### Example Status

```yaml
status:
  conditions:
    - type: Ready
      status: "True"
      lastTransitionTime: "2026-04-01T12:00:00Z"
      reason: Valid
      message: "Client validated and cluster 'production' exists"
```

## Rendered Output

A `RadiusClient` named `core-switch` produces the following in `clients.conf`:

```
client core-switch {
    ipaddr = 10.0.1.0/24
    secret = ${file:/etc/freeradius/secrets/switch-secret/shared-secret}
    nastype = cisco
    # location = building-a-floor-3
    # role = distribution
}
```

## Multiple Clients Per Cluster

You can define as many `RadiusClient` resources as needed for a single cluster. All clients with the same `clusterRef` are aggregated into a single `clients.conf`:

```bash
kubectl get radiusclients -n radius -l clusterRef=production
```

When any `RadiusClient` is created, updated, or deleted, the operator re-renders the full configuration and triggers a rolling update of the associated `RadiusCluster`.
