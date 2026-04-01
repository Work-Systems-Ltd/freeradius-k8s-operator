# Development

Build, test, and contribute to the FreeRADIUS Kubernetes Operator.

---

## Prerequisites

| Tool | Version | Purpose |
|:-----|:--------|:--------|
| Go | 1.22+ | Build the operator |
| Docker | 20+ | Build container images |
| kind | 0.20+ | Local Kubernetes cluster |
| kubectl | 1.28+ | Cluster interaction |
| controller-gen | latest | CRD and RBAC generation |
| golangci-lint | latest | Linting |

## Project Structure

```
├── api/v1alpha1/          # CRD type definitions and validation
├── cmd/operator/          # Entrypoint (main.go)
├── internal/
│   ├── controller/        # Reconciliation logic for all three CRDs
│   ├── renderer/          # Pure-function config generation
│   │   └── templates/     # Go text/template files
│   ├── status/            # Status condition management
│   └── metrics/           # Prometheus metric definitions
├── config/
│   ├── crd/               # Generated CRD YAML manifests
│   └── rbac/              # ClusterRole and binding
├── dev/                   # kind cluster config and kubeconfig
├── e2e/                   # End-to-end test suite
├── Dockerfile             # Two-stage distroless build
└── Makefile               # Build, test, lint, generate targets
```

## Build

```bash
# Compile the operator binary
make build

# Output: bin/freeradius-operator
```

## Code Generation

After modifying CRD types in `api/v1alpha1/`, regenerate manifests:

```bash
# Generate DeepCopy methods
make generate

# Generate CRD and RBAC YAML
make manifests
```

Always commit the regenerated files alongside your type changes.

## Testing

### Run all tests

```bash
make test
```

### Test categories

The project uses three testing strategies:

**Unit tests** — Standard Go tests with [testify](https://github.com/stretchr/testify) assertions. Cover individual functions and methods.

**Property-based tests** — Use [pgregory.net/rapid](https://pkg.go.dev/pgregory.net/rapid) to verify correctness across randomly generated inputs. Each property runs 100+ iterations. Key properties tested:

- Deterministic rendering (same inputs produce same outputs)
- Secret isolation (plaintext values never leak into config)
- Valid FreeRADIUS syntax in rendered output
- Idempotent reconciliation

**Golden file tests** — Snapshot tests that compare rendered configuration against known-good baselines in `internal/renderer/testdata/`. Update golden files when you intentionally change rendering behavior:

```bash
# Update golden files after intentional rendering changes
UPDATE_GOLDEN=1 go test ./internal/renderer/...
```

### Run specific test packages

```bash
# Renderer tests only
go test ./internal/renderer/...

# Controller tests only
go test ./internal/controller/...

# Validation tests
go test ./api/v1alpha1/...
```

## Linting

```bash
make lint
```

This runs [golangci-lint](https://golangci-lint.run/) with the project's configuration.

## Local Development with kind

### Set up the cluster

```bash
# Create a kind cluster with RADIUS ports exposed
kind create cluster \
  --name freeradius-dev \
  --config dev/kind-config.yaml \
  --kubeconfig dev/kubeconfig \
  --wait 120s

export KUBECONFIG=$(pwd)/dev/kubeconfig
```

The kind configuration exposes:
- Port **30812** (UDP) — RADIUS authentication (1812)
- Port **30813** (UDP) — RADIUS accounting (1813)

### Build, load, and run

```bash
# Install CRDs
kubectl apply -f config/crd/

# Build the image and load it into kind (no registry needed)
make load-image

# Run the operator
make dev-run
```

### Iterate

The typical development loop:

1. Edit code
2. `make build && make load-image`
3. Restart the operator: `make dev-run`
4. Apply test resources: `kubectl apply -f examples/`
5. Verify: `kubectl get rc,rcl,rp -A`

### Tear down

```bash
kind delete cluster --name freeradius-dev
```

## Adding a New Module Type

To add support for a new FreeRADIUS module:

1. **Define the config struct** in `api/v1alpha1/radiuscluster_types.go`
2. **Add the type enum value** to the `ModuleConfig.Type` field
3. **Create a template** in `internal/renderer/templates/mods-enabled/`
4. **Add rendering logic** in `internal/renderer/modules.go`
5. **Write tests** — unit test, property test, and a golden file
6. **Regenerate**: `make generate && make manifests`

## Architecture Decisions

| Decision | Rationale |
|:---------|:----------|
| Pure-function renderer | Stateless, deterministic, easy to test in isolation |
| Status subresource | Avoids optimistic concurrency conflicts with spec updates |
| Cross-controller enqueueing | Client/Policy changes trigger cluster reconciliation without tight coupling |
| Template-driven config | Go templates are readable and map closely to FreeRADIUS config syntax |
| Property-based testing | Catches edge cases that example-based tests miss |
| Distroless base image | Minimal attack surface — no shell, no package manager |
