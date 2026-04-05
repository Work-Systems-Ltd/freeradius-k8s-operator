# Basic Example — SQL Authentication

A minimal FreeRADIUS deployment using PostgreSQL for user authentication.

## Resources
- **secrets.yaml** — Database and RADIUS shared secret credentials
- **radiuscluster.yaml** — 2-replica FreeRADIUS cluster with the SQL module
- **radiusclient.yaml** — A NAS client (Cisco switch on 10.0.1.0/24)
- **radiuspolicy.yaml** — Post-auth policy that assigns VLANs to admin users

## Deploy
```sh
kubectl apply -f example/basic/
```
