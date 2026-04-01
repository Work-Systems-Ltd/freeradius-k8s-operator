---
title: Upgrading
parent: Guides
nav_order: 6
---

# Upgrading
{: .no_toc }

How to upgrade the operator, FreeRADIUS, and CRDs safely.
{: .fs-6 .fw-300 }

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}

---

## Upgrade Types

There are three independent things you might upgrade:

| Component | What Changes | Risk Level |
|:----------|:-------------|:-----------|
| **FreeRADIUS image** | The RADIUS server binary and libraries | Low — rolling update, no config change |
| **Operator** | The controller binary that manages resources | Medium — new reconciliation logic |
| **CRDs** | The schema definitions for custom resources | High — may require manifest changes |

---

## Upgrading FreeRADIUS

Change the `spec.image` on your `RadiusCluster`:

```bash
kubectl patch radiuscluster production -n radius \
  --type merge \
  -p '{"spec":{"image":"freeradius/freeradius-server:3.2.4"}}'
```

**What happens**:
1. The operator detects the image change
2. Updates the Deployment's pod template
3. Kubernetes performs a rolling update (`maxUnavailable: 0`, `maxSurge: 1`)
4. At least one pod serves traffic throughout the upgrade

**Pre-flight checks**:
- Review the FreeRADIUS [changelog](https://github.com/FreeRADIUS/freeradius-server/blob/master/doc/ChangeLog) for breaking changes
- Test the new image in a non-production cluster first
- Verify your modules are compatible with the new version

**Rollback**:

```bash
kubectl patch radiuscluster production -n radius \
  --type merge \
  -p '{"spec":{"image":"freeradius/freeradius-server:3.2.3"}}'
```

---

## Upgrading the Operator

### 1. Check release notes

Review the release notes for any breaking changes, new features, or required CRD updates.

### 2. Update CRDs first

Always update CRDs before the operator. New operator versions may expect fields that don't exist in older CRD schemas.

```bash
kubectl apply -f config/crd/
```

Verify:

```bash
kubectl get crds -o custom-columns=NAME:.metadata.name,AGE:.metadata.creationTimestamp \
  | grep radius.operator.io
```

### 3. Update the operator

**If running as a Deployment:**

```bash
kubectl set image deployment/freeradius-operator \
  operator=freeradius-operator:<new-version> \
  -n freeradius-system
```

**If running locally:**

```bash
git pull
make build
make dev-run
```

### 4. Verify reconciliation

After the new operator starts, it reconciles all existing resources:

```bash
# Watch for reconciliation
kubectl get radiusclusters -A -w

# Check operator logs for errors
kubectl logs -n freeradius-system deploy/freeradius-operator --since=5m | grep -i error
```

---

## Upgrading CRDs

CRD changes can add new fields, remove deprecated fields, or change validation rules.

### Adding new fields

New optional fields are backwards-compatible. Existing resources continue to work — they simply don't use the new field until you update them.

```bash
kubectl apply -f config/crd/
# Existing resources are unaffected
```

### Schema changes

If a field is renamed, moved, or its type changes, you need to update your manifests:

1. Back up your current resources:
   ```bash
   kubectl get radiusclusters -A -o yaml > backup-clusters.yaml
   kubectl get radiusclients -A -o yaml > backup-clients.yaml
   kubectl get radiuspolicies -A -o yaml > backup-policies.yaml
   ```
2. Apply the new CRDs:
   ```bash
   kubectl apply -f config/crd/
   ```
3. Update your manifests to match the new schema
4. Re-apply:
   ```bash
   kubectl apply -f updated-manifests/
   ```

{: .warning }
> The project is currently at `v1alpha1`. The API surface may change between releases. Always review release notes and back up resources before upgrading CRDs.

---

## Zero-Downtime Checklist

For production upgrades:

- [ ] Review release notes and changelog
- [ ] Back up all RADIUS custom resources (`kubectl get <type> -A -o yaml`)
- [ ] Test the upgrade in a staging environment
- [ ] Update CRDs first, then the operator
- [ ] Monitor operator logs during the first reconciliation cycle
- [ ] Verify `Available=True` on all RadiusClusters after upgrade
- [ ] Run a smoke test (e.g., `radtest`) against the RADIUS service
- [ ] Confirm Prometheus metrics are being scraped
