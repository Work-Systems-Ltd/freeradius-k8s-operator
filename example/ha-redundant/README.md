# HA Redundant Example

High-availability FreeRADIUS deployment with redundant SQL backends. If the
primary database is unreachable, FreeRADIUS automatically falls back to the
replica.

## Features
- 3 replicas with HPA (scales to 10)
- PDB with minAvailable=2
- Redundant SQL modules (primary + replica failover)

## Resources
- **secrets.yaml** — Primary/replica DB passwords and NAS shared secret
- **radiuscluster.yaml** — HA cluster with two SQL modules, autoscaling, and PDB
- **radiusclient.yaml** — Entire 10.0.0.0/8 datacenter network
- **radiuspolicy.yaml** — Redundant SQL calls for authorize and accounting stages

## Deploy
```sh
kubectl apply -f example/ha-redundant/
```
