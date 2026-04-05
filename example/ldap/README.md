# LDAP Example

Authenticates users against an LDAP directory (e.g., Active Directory or OpenLDAP).

## Resources
- **secrets.yaml** — LDAP bind password and NAS shared secret
- **radiuscluster.yaml** — Cluster with rlm_ldap module
- **radiusclient.yaml** — Office switch subnet
- **radiuspolicy.yaml** — Policies calling LDAP in authorize and authenticate stages

## Deploy
```sh
kubectl apply -f example/ldap/
```
