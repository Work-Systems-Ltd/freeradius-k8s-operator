# Split-Mode Example

Deploys auth, accounting, and CoA as independent Deployments and Services
so each function scales independently.

## Features
- Auth: LoadBalancer with HPA (3-20 replicas)
- Accounting: ClusterIP with 2 replicas
- CoA: ClusterIP with 1 replica on port 3799
- Each function gets its own Service IP

## Resources
- **secrets.yaml** — Database and NAS credentials
- **radiuscluster.yaml** — Split-mode cluster with per-function scaling
- **radiusclient.yaml** — Edge router subnet

## Deploy
```sh
kubectl apply -f example/split-mode/
```
