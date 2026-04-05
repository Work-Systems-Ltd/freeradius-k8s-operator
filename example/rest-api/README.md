# REST API Example

Authenticates users via an external REST API. FreeRADIUS forwards authorize,
authenticate, and accounting requests to an HTTP backend.

## Resources
- **secrets.yaml** — API bearer token and NAS shared secret
- **radiuscluster.yaml** — Cluster with rlm_rest module pointing at an auth API
- **radiusclient.yaml** — Wireless AP subnet
- **radiuspolicy.yaml** — Policies that call the REST module in authorize and authenticate stages

## Deploy
```sh
kubectl apply -f example/rest-api/
```
