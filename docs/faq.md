---
title: FAQ
nav_order: 9
---

# Frequently Asked Questions
{: .no_toc }

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}

---

## General

### What version of FreeRADIUS does this operator support?

The operator generates configuration compatible with FreeRADIUS 3.x. Specify the exact version via the `spec.image` field:

```yaml
spec:
  image: freeradius/freeradius-server:3.2.3
```

FreeRADIUS 4.x support is not yet available. The configuration syntax differs significantly between 3.x and 4.x.

### Can I run multiple RadiusClusters in the same namespace?

Yes. Each `RadiusCluster` creates its own independent Deployment, Service, ConfigMap, and optional HPA. `RadiusClient` and `RadiusPolicy` resources are associated with a specific cluster via `clusterRef`.

### Can a RadiusClient or RadiusPolicy reference a cluster in a different namespace?

No. The `clusterRef` is resolved within the same namespace. For multi-tenant deployments, use separate namespaces with their own `RadiusCluster` instances. See the [Multi-Tenant Isolation example](/freeradius-k8s-operator/examples/#multi-tenant-isolation).

### Is this production-ready?

The project is in alpha (`v1alpha1`). The operator is functional and tested (unit, property-based, golden file, and e2e tests), but the API surface may change between releases. It is suitable for development, testing, and non-critical production workloads.

---

## Configuration

### How do I add a custom FreeRADIUS configuration file?

The operator generates all configuration from CRD specs. There is no mechanism to inject raw configuration files. If you need functionality not covered by the CRDs, consider:

1. Opening a feature request for a new module type or policy action
2. Using a `RadiusPolicy` with a `call` action to invoke a module that handles your custom logic
3. Building a custom FreeRADIUS image with your configuration baked in (less flexible, but works as a short-term solution)

### How do I update FreeRADIUS?

Change the `spec.image` field on your `RadiusCluster`:

```bash
kubectl patch radiuscluster production -n radius \
  --type merge \
  -p '{"spec":{"image":"freeradius/freeradius-server:3.2.4"}}'
```

The operator triggers a rolling update. The `maxUnavailable: 0` strategy ensures at least one pod serves traffic throughout the upgrade.

### Can I use a custom FreeRADIUS image?

Yes. The operator doesn't require a specific image — it only needs a container that runs FreeRADIUS and reads configuration from `/etc/freeradius/`. You can use a custom image with additional modules, dictionaries, or scripts.

### How are configuration files structured in the ConfigMap?

Files are stored with a flat key scheme — directory separators (`/`) are replaced with `__`:

| ConfigMap Key | Actual File Path |
|:-------------|:----------------|
| `radiusd.conf` | `/etc/freeradius/radiusd.conf` |
| `clients.conf` | `/etc/freeradius/clients.conf` |
| `mods-enabled__sql` | `/etc/freeradius/mods-enabled/sql` |
| `sites-enabled__default` | `/etc/freeradius/sites-enabled/default` |

An init container reconstructs the directory structure before FreeRADIUS starts.

---

## Operations

### How does the operator handle Secret changes?

When a Kubernetes Secret referenced by a `RadiusCluster` (directly or via a client/module) is updated, the operator detects the change during its next reconciliation and triggers a rolling update. The reconciliation loop runs every 30 seconds for resources in a retry state, or immediately when a watched resource changes.

### What happens when I delete a RadiusCluster?

All owned resources (Deployment, Service, ConfigMap, HPA) are garbage collected by Kubernetes via owner references. Pods terminate gracefully. `RadiusClient` and `RadiusPolicy` resources that reference the deleted cluster are **not** automatically deleted — they'll show `Invalid` status until you either delete them or create a new cluster with the same name.

### Can I scale to zero?

No. The `replicas` field has a minimum of 1. If you need to completely stop RADIUS service, delete the `RadiusCluster` resource.

### How do I check what configuration FreeRADIUS is running?

Inspect the ConfigMap directly:

```bash
# List all config keys
kubectl get configmap <cluster-name>-config -n <namespace> -o jsonpath='{.data}' | jq 'keys'

# View a specific file
kubectl get configmap <cluster-name>-config -n <namespace> \
  -o jsonpath='{.data.clients\.conf}'
```

### Does the operator support FreeRADIUS clustering (replicated sessions)?

The operator deploys multiple FreeRADIUS replicas behind a Kubernetes Service, but it does not configure FreeRADIUS's native clustering or session replication. For session-aware features (like simultaneous-use checks), use the Redis module as a shared session store.

---

## Development

### How do I run the tests?

```bash
make test
```

This runs unit tests, property-based tests, and golden file tests. E2E tests require a running Kubernetes cluster (kind).

### How do I update golden files after changing rendering?

```bash
UPDATE_GOLDEN=1 go test ./internal/renderer/...
```

Review the diff carefully before committing — golden file changes represent actual configuration output changes.

### How do I add a new CRD field?

1. Edit the type definition in `api/v1alpha1/`
2. Run `make generate` (DeepCopy methods)
3. Run `make manifests` (CRD YAML)
4. Update the renderer and/or controller logic
5. Add tests
6. Commit both the code changes and the regenerated files
