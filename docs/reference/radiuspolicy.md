---
title: RadiusPolicy
parent: CRD Reference
nav_order: 3
---

# RadiusPolicy
{: .no_toc }

Defines authentication and authorization logic as declarative rules.
{: .fs-6 .fw-300 }

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}

---

## Overview

A `RadiusPolicy` maps to `unlang` policy logic within a FreeRADIUS virtual server. Instead of writing raw unlang, you declare match conditions and actions — the operator renders the corresponding `if` blocks in the appropriate processing stage.

**API Group**: `radius.operator.io/v1alpha1`
**Kind**: `RadiusPolicy`
**Short Name**: `rp`
**Scope**: Namespaced

## Full Example

```yaml
apiVersion: radius.operator.io/v1alpha1
kind: RadiusPolicy
metadata:
  name: vlan-assignment
  namespace: radius
spec:
  clusterRef: production
  stage: post-auth
  priority: 200
  match:
    all:
      - attribute: NAS-IP-Address
        operator: "=="
        value: "10.0.1.1"
      - attribute: User-Name
        operator: "=~"
        value: "^admin-.*"
  actions:
    - type: set
      attribute: Tunnel-Type
      value: VLAN
    - type: set
      attribute: Tunnel-Medium-Type
      value: IEEE-802
    - type: set
      attribute: Tunnel-Private-Group-Id
      value: "100"
```

## Spec Fields

### `clusterRef` (required)

Name of the `RadiusCluster` this policy applies to. Must be in the same namespace.

```yaml
clusterRef: production
```

### `stage` (required)

The FreeRADIUS processing stage where this policy is evaluated.

```yaml
stage: authorize
```

| Type | Allowed Values |
|:-----|:---------------|
| enum | `authorize`, `authenticate`, `preacct`, `accounting`, `post-auth`, `pre-proxy`, `post-proxy`, `session` |

**Stage descriptions**:

| Stage | When It Runs |
|:------|:-------------|
| `authorize` | Before authentication — used to look up user data and set Auth-Type |
| `authenticate` | During credential verification |
| `preacct` | Before accounting processing |
| `accounting` | During accounting record processing |
| `post-auth` | After successful authentication — used for VLAN assignment, attribute injection |
| `pre-proxy` | Before proxying a request to another RADIUS server |
| `post-proxy` | After receiving a proxied response |
| `session` | Session management (simultaneous-use checks) |

### `priority` (required)

Sort order within a stage. Lower values are evaluated first. Policies with the same priority have no guaranteed order.

```yaml
priority: 100
```

| Type | Default |
|:-----|:--------|
| int32 | — |

{: .tip }
> Use gaps in priority values (100, 200, 300) to leave room for inserting policies later without renumbering.

### `match`

Condition tree that determines when this policy's actions execute. Supports boolean composition with `all` (AND), `any` (OR), and `none` (NOT).

```yaml
match:
  all:
    - attribute: User-Name
      operator: "=="
      value: "admin"
    - attribute: NAS-IP-Address
      operator: "!="
      value: "10.0.0.1"
  any:
    - attribute: Called-Station-Id
      operator: "=~"
      value: "^AA-BB-.*"
  none:
    - attribute: Service-Type
      operator: "=="
      value: "Call-Check"
```

#### Match Composition

| Field | Behavior |
|:------|:---------|
| `all` | All conditions must be true (logical AND) |
| `any` | At least one condition must be true (logical OR) |
| `none` | No condition may be true (logical NOT) |

When multiple fields are specified, they are combined with AND: the `all` block AND the `any` block AND the `none` block must all be satisfied.

#### MatchLeaf

Each leaf condition compares a RADIUS attribute against a value.

| Field | Type | Description |
|:------|:-----|:------------|
| `attribute` | string | RADIUS attribute name (e.g., `User-Name`, `NAS-IP-Address`) |
| `operator` | string | Comparison operator |
| `value` | string | Value to compare against |

**Supported operators**:

| Operator | Meaning |
|:---------|:--------|
| `==` | Equal |
| `!=` | Not equal |
| `>` | Greater than |
| `>=` | Greater than or equal |
| `<` | Less than |
| `<=` | Less than or equal |
| `=~` | Regex match |
| `!~` | Regex not match |

### `actions`

List of actions to execute when the match conditions are satisfied. Actions are executed in order.

```yaml
actions:
  - type: set
    attribute: Reply-Message
    value: "Welcome, admin"
  - type: accept
```

#### PolicyAction

| Field | Type | Description |
|:------|:-----|:------------|
| `type` | enum | `set`, `call`, `reject`, `accept` |
| `module` | string | Module name (only for `call` type) |
| `attribute` | string | RADIUS attribute (only for `set` type) |
| `value` | string | Attribute value (only for `set` type) |

**Action types**:

| Type | Description | Required Fields |
|:-----|:------------|:----------------|
| `set` | Set a RADIUS reply attribute | `attribute`, `value` |
| `call` | Invoke a named module | `module` |
| `reject` | Reject the request immediately | — |
| `accept` | Accept the request immediately | — |

## Status

### Conditions

| Type | Description |
|:-----|:------------|
| `Ready` | Policy is valid and the referenced cluster exists |
| `Invalid` | Validation failed — unknown stage, invalid action type, or missing clusterRef |

## Rendered Output

The example policy above renders as unlang in `sites-enabled/default`:

```
post-auth {
    # Policy: vlan-assignment (priority: 200)
    if ((NAS-IP-Address == "10.0.1.1") && (User-Name =~ "^admin-.*")) {
        update reply {
            Tunnel-Type := VLAN
            Tunnel-Medium-Type := IEEE-802
            Tunnel-Private-Group-Id := "100"
        }
    }
}
```

## Common Patterns

### Reject requests from unknown domains

```yaml
spec:
  stage: authorize
  priority: 50
  match:
    none:
      - attribute: User-Name
        operator: "=~"
        value: "@example\\.com$"
  actions:
    - type: reject
```

### Call an SQL module for accounting

```yaml
spec:
  stage: accounting
  priority: 100
  match:
    all:
      - attribute: Acct-Status-Type
        operator: "=="
        value: "Start"
  actions:
    - type: call
      module: sql
```

### Set VLAN based on user group

```yaml
spec:
  stage: post-auth
  priority: 300
  match:
    any:
      - attribute: User-Name
        operator: "=~"
        value: "^guest-"
  actions:
    - type: set
      attribute: Tunnel-Type
      value: VLAN
    - type: set
      attribute: Tunnel-Private-Group-Id
      value: "200"
```
