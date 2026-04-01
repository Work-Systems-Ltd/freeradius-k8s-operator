# Contributing to freeradius-k8s-operator

Thanks for your interest in contributing! This document covers the basics of how to get involved.

## Getting Started

1. Fork the repository
2. Clone your fork:
   ```bash
   git clone https://github.com/<your-username>/freeradius-k8s-operator.git
   cd freeradius-k8s-operator
   ```
3. Set up your development environment (see [Development docs](https://tbotnz.github.io/freeradius-k8s-operator/development/))

## Development Workflow

```bash
# Create a feature branch
git checkout -b feature/my-change

# Make your changes, then run tests and lint
make test
make lint

# If you changed CRD types in api/v1alpha1/
make generate
make manifests

# Commit and push
git add -A
git commit -m "feat: description of change"
git push origin feature/my-change
```

## What to Contribute

### Good first issues

- Adding test coverage for edge cases
- Improving error messages in status conditions
- Documentation fixes and improvements

### Larger contributions

- New module types (add to `api/v1alpha1/`, renderer, and templates)
- New policy action types
- Performance improvements in the reconciliation loop

For larger changes, please open an issue first to discuss the approach.

## Code Standards

- **Tests required** — All changes should include tests. The project uses three testing strategies:
  - Unit tests with [testify](https://github.com/stretchr/testify)
  - Property-based tests with [pgregory.net/rapid](https://pkg.go.dev/pgregory.net/rapid)
  - Golden file tests for rendered configuration output
- **Lint clean** — `make lint` must pass
- **Generated files** — If you change types in `api/v1alpha1/`, commit the regenerated files from `make generate && make manifests` alongside your code changes
- **Secrets** — Never log, embed in ConfigMaps, or include plaintext secret values in rendered configuration. Always use `SecretRef` and file-path references.

## Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
feat: add redis module support
fix: handle missing secret gracefully in reconciler
docs: update CRD reference for RadiusPolicy
test: add property tests for LDAP module rendering
refactor: extract template helpers into separate file
```

## Pull Requests

- Keep PRs focused — one logical change per PR
- Include a description of what changed and why
- Reference any related issues
- Ensure CI passes (tests + lint)

## Reporting Issues

Open an issue on [GitHub](https://github.com/tbotnz/freeradius-k8s-operator/issues) with:

- What you expected to happen
- What actually happened
- Steps to reproduce
- Operator version / commit hash
- Kubernetes version (`kubectl version`)
- Relevant logs and resource YAML (redact secrets)
