# Troubleshooting

Common issues and how to resolve them.

---

## Diagnosing Issues

### Check resource status

The first step for any issue is to check the status conditions on your resources:

```bash
# RadiusCluster status
kubectl get radiuscluster <name> -n <namespace> -o yaml

# RadiusClient status
kubectl get radiusclient <name> -n <namespace> -o yaml

# All RADIUS resources at a glance
kubectl get rc,rcl,rp -n <namespace>
```

### Check operator logs

```bash
# If running as a Deployment
kubectl logs -n freeradius-system deploy/freeradius-operator -f

# If running locally
# Logs appear in the terminal where you ran `make dev-run`
```

### Check events

The operator emits Kubernetes Events for significant state changes:

```bash
kubectl get events -n <namespace> --sort-by='.lastTimestamp' | grep radius
```

---

## Common Issues

### RadiusCluster stuck in `Progressing`

**Symptom**: The cluster's `Progressing` condition stays `True` and `Available` never becomes `True`.

**Causes**:
1. **FreeRADIUS pods failing to start** — Check pod logs:
   ```bash
   kubectl logs -n <namespace> -l app.kubernetes.io/name=<cluster-name>
   ```
2. **Image pull failure** — Verify the image exists and is accessible:
   ```bash
   kubectl describe pod -n <namespace> -l app.kubernetes.io/name=<cluster-name>
   ```
   Look for `ImagePullBackOff` or `ErrImagePull` events.
3. **Insufficient resources** — The cluster may not have enough CPU/memory to schedule pods:
   ```bash
   kubectl describe pod -n <namespace> -l app.kubernetes.io/name=<cluster-name> | grep -A 5 Events
   ```

### RadiusCluster shows `Degraded`

**Symptom**: The `Degraded` condition is `True` with reason `MissingSecret`.

**Cause**: A Secret referenced by the cluster, a client, or a module does not exist.

**Fix**: Create the missing Secret. Check the condition message for the exact Secret name:

```bash
kubectl get radiuscluster <name> -n <namespace> \
  -o jsonpath='{.status.conditions[?(@.type=="Degraded")].message}'
```

Then create it:

```bash
kubectl create secret generic <secret-name> \
  --namespace=<namespace> \
  --from-literal=<key>=<value>
```

The operator requeues every 30 seconds and will reconcile automatically once the Secret exists.

### RadiusClient shows `Invalid`

**Symptom**: The `Invalid` condition is `True`.

**Common causes**:

| Message | Fix |
|:--------|:----|
| Cluster `<name>` not found | Create the referenced `RadiusCluster` first, or fix the `clusterRef` |
| Invalid IP format | Ensure `spec.ip` is a valid IPv4, IPv6, or CIDR (e.g., `10.0.1.1`, `10.0.0.0/24`, `2001:db8::1`) |

### RadiusPolicy shows `Invalid`

**Symptom**: The `Invalid` condition is `True`.

**Common causes**:

| Message | Fix |
|:--------|:----|
| Unknown stage | Use one of: `authorize`, `authenticate`, `preacct`, `accounting`, `post-auth`, `pre-proxy`, `post-proxy`, `session` |
| Invalid action type | Use one of: `set`, `call`, `reject`, `accept` |
| Cluster `<name>` not found | Create the referenced `RadiusCluster` first |

### ConfigMap not updating after changes

**Symptom**: You changed a `RadiusClient` or `RadiusPolicy` but the ConfigMap still has the old content.

**Checks**:
1. Verify the `clusterRef` matches the `RadiusCluster` name exactly (case-sensitive)
2. Check that the client/policy is in the **same namespace** as the cluster
3. Look at operator logs for reconciliation errors

```bash
# Verify clusterRef
kubectl get radiusclient <name> -n <namespace> -o jsonpath='{.spec.clusterRef}'

# Check if reconciliation ran
kubectl logs -n freeradius-system deploy/freeradius-operator | grep <cluster-name>
```

### Pods not restarting after config change

**Symptom**: The ConfigMap updated but pods are still running with the old configuration.

**Explanation**: The operator triggers a rolling update by annotating the pod template with a config hash. If the Deployment controller is paused or the cluster has scheduling issues, new pods may not roll out.

```bash
# Check rollout status
kubectl rollout status deployment/<cluster-name> -n <namespace>

# Force a restart if needed
kubectl rollout restart deployment/<cluster-name> -n <namespace>
```

### HPA not scaling

**Symptom**: `autoscaling.enabled: true` but pods stay at `minReplicas`.

**Checks**:
1. **Metrics Server installed?**
   ```bash
   kubectl top pods -n <namespace>
   ```
   If this fails, install [Metrics Server](https://github.com/kubernetes-sigs/metrics-server).

2. **CPU requests set?** The HPA needs `resources.requests.cpu` to calculate utilization:
   ```yaml
   resources:
     requests:
       cpu: 250m  # Required
   ```

3. **Check HPA status**:
   ```bash
   kubectl get hpa -n <namespace>
   kubectl describe hpa <cluster-name> -n <namespace>
   ```

### Authentication failing (`radtest` returns Access-Reject)

**Symptom**: FreeRADIUS is running but rejects all authentication attempts.

**Checks**:
1. **Client IP matches?** The requesting device's IP must match a `RadiusClient` `spec.ip`:
   ```bash
   kubectl get radiusclients -n <namespace> -o wide
   ```
2. **Shared secret matches?** The secret used by `radtest` must match the value in the Kubernetes Secret:
   ```bash
   kubectl get secret <secret-name> -n <namespace> -o jsonpath='{.data.<key>}' | base64 -d
   ```
3. **Check FreeRADIUS logs** for detailed reject reasons:
   ```bash
   kubectl logs -n <namespace> <pod-name>
   ```

---

## Operator Not Starting

### RBAC errors

**Symptom**: Operator logs show `forbidden` errors when accessing Kubernetes resources.

**Fix**: Apply the RBAC manifests:

```bash
kubectl apply -f config/rbac/role.yaml
kubectl apply -f config/rbac/role_binding.yaml
```

Ensure the ServiceAccount, ClusterRole, and ClusterRoleBinding all reference the correct names and namespace.

### CRDs not found

**Symptom**: Operator logs show errors about unknown resource types.

**Fix**: Install or re-install the CRDs:

```bash
kubectl apply -f config/crd/
kubectl get crds | grep radius.operator.io
```

---

## Getting Help

If you've worked through this guide and are still stuck:

1. Check the [GitHub Issues](https://github.com/Work-Systems-Ltd/freeradius-k8s-operator/issues) for similar problems
2. Open a new issue with:
   - Operator version / commit hash
   - Kubernetes version (`kubectl version`)
   - The full resource YAML (redact secrets)
   - Operator logs around the time of the issue
   - Output of `kubectl describe` for the affected resource
