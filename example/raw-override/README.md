# Raw Override Example

Demonstrates the rawConfig escape hatch for FreeRADIUS configuration that
can't be expressed through the CRD's typed fields.

## Use cases
- Custom module types without dedicated CRD support (e.g., rlm_python)
- Complex unlang logic that goes beyond the policy CRD's match/action model
- Vendor-specific FreeRADIUS directives

## Resources
- **radiuscluster.yaml** — Cluster with a raw Python module config
- **radiuspolicy.yaml** — Raw unlang for MAC authentication bypass

## Deploy
```sh
kubectl apply -f example/raw-override/
```
