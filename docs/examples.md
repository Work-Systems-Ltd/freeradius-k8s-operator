# Examples

Complete deployment scenarios you can adapt to your environment. Ready-to-apply manifests are in the [`example/`](https://github.com/Work-Systems-Ltd/freeradius-k8s-operator/tree/master/example) directory:

| Example | Description |
|:--------|:------------|
| [`basic/`](https://github.com/Work-Systems-Ltd/freeradius-k8s-operator/tree/master/example/basic) | SQL-backed auth with VLAN assignment policy |
| [`rest-api/`](https://github.com/Work-Systems-Ltd/freeradius-k8s-operator/tree/master/example/rest-api) | Authentication via an external REST API |
| [`ldap/`](https://github.com/Work-Systems-Ltd/freeradius-k8s-operator/tree/master/example/ldap) | LDAP / Active Directory authentication |
| [`ha-redundant/`](https://github.com/Work-Systems-Ltd/freeradius-k8s-operator/tree/master/example/ha-redundant) | Redundant SQL failover with autoscaling and PDB |
| [`split-mode/`](https://github.com/Work-Systems-Ltd/freeradius-k8s-operator/tree/master/example/split-mode) | Independent scaling for auth, accounting, and CoA |
| [`raw-override/`](https://github.com/Work-Systems-Ltd/freeradius-k8s-operator/tree/master/example/raw-override) | Escape hatch for custom modules and raw unlang |

```bash
# Deploy the basic example
kubectl apply -f config/crd/
kubectl apply -f example/basic/
```

The scenarios below show more advanced patterns.

---

## Campus Wireless with LDAP

A university or corporate campus with 802.1X wireless authentication against Active Directory. EAP-PEAP handles the authentication method, LDAP resolves user accounts, and SQL stores accounting records.

### Secrets

```bash
# Active Directory bind password
kubectl create secret generic ad-bind-password \
  --namespace=radius \
  --from-literal=password='s3cureBindP@ss'

# SQL database credentials
kubectl create secret generic radius-db-creds \
  --namespace=radius \
  --from-literal=password='dbP@ssw0rd'

# EAP TLS certificate
kubectl create secret tls eap-server-cert \
  --namespace=radius \
  --cert=radius-eap.crt \
  --key=radius-eap.key

# Shared secret for all wireless controllers
kubectl create secret generic wlc-secret \
  --namespace=radius \
  --from-literal=shared-secret='W1r3l3ssK3y!'
```

### RadiusCluster

```yaml
apiVersion: radius.operator.io/v1alpha1
kind: RadiusCluster
metadata:
  name: campus-wifi
  namespace: radius
spec:
  image: freeradius/freeradius-server:3.2.3
  replicas: 3
  resources:
    requests:
      cpu: 500m
      memory: 512Mi
    limits:
      cpu: "2"
      memory: 1Gi
  autoscaling:
    enabled: true
    minReplicas: 3
    maxReplicas: 8
    targetCPUUtilizationPercentage: 70
  modules:
    - name: ldap
      type: ldap
      enabled: true
      ldap:
        server: ldaps://ad.campus.edu
        port: 636
        baseDN: "dc=campus,dc=edu"
        identity: "cn=radius-svc,ou=service-accounts,dc=campus,dc=edu"
        credentialsRef:
          name: ad-bind-password
          key: password
    - name: eap
      type: eap
      enabled: true
      eap:
        defaultMethod: peap
        tls:
          certRef:
            name: eap-server-cert
            key: tls.crt
          keyRef:
            name: eap-server-cert
            key: tls.key
    - name: sql
      type: sql
      enabled: true
      sql:
        driver: postgresql
        server: radius-db.campus.edu
        port: 5432
        database: radius_acct
        credentialsRef:
          name: radius-db-creds
          key: password
```

### RadiusClients — Wireless Controllers

```yaml
apiVersion: radius.operator.io/v1alpha1
kind: RadiusClient
metadata:
  name: wlc-building-a
  namespace: radius
spec:
  clusterRef: campus-wifi
  ip: 10.1.1.10
  secretRef:
    name: wlc-secret
    key: shared-secret
  nasType: cisco
  metadata:
    location: building-a-mdf
    model: C9800-40
---
apiVersion: radius.operator.io/v1alpha1
kind: RadiusClient
metadata:
  name: wlc-building-b
  namespace: radius
spec:
  clusterRef: campus-wifi
  ip: 10.1.2.10
  secretRef:
    name: wlc-secret
    key: shared-secret
  nasType: cisco
  metadata:
    location: building-b-mdf
    model: C9800-40
---
apiVersion: radius.operator.io/v1alpha1
kind: RadiusClient
metadata:
  name: wlc-library
  namespace: radius
spec:
  clusterRef: campus-wifi
  ip: 10.1.3.10
  secretRef:
    name: wlc-secret
    key: shared-secret
  nasType: cisco
  metadata:
    location: library-idf-2
    model: C9800-L
```

### RadiusPolicies

```yaml
# Reject users not in the campus domain
apiVersion: radius.operator.io/v1alpha1
kind: RadiusPolicy
metadata:
  name: require-campus-domain
  namespace: radius
spec:
  clusterRef: campus-wifi
  stage: authorize
  priority: 50
  match:
    none:
      - attribute: User-Name
        operator: "=~"
        value: "@campus\\.edu$"
  actions:
    - type: reject
---
# Assign staff to VLAN 100
apiVersion: radius.operator.io/v1alpha1
kind: RadiusPolicy
metadata:
  name: staff-vlan
  namespace: radius
spec:
  clusterRef: campus-wifi
  stage: post-auth
  priority: 100
  match:
    all:
      - attribute: User-Name
        operator: "=~"
        value: "^staff-"
  actions:
    - type: set
      attribute: Tunnel-Type
      value: VLAN
    - type: set
      attribute: Tunnel-Medium-Type
      value: IEEE-802
    - type: set
      attribute: Tunnel-Private-Group-Id
      value: "100"
---
# Assign students to VLAN 200
apiVersion: radius.operator.io/v1alpha1
kind: RadiusPolicy
metadata:
  name: student-vlan
  namespace: radius
spec:
  clusterRef: campus-wifi
  stage: post-auth
  priority: 200
  match:
    all:
      - attribute: User-Name
        operator: "=~"
        value: "^student-"
  actions:
    - type: set
      attribute: Tunnel-Type
      value: VLAN
    - type: set
      attribute: Tunnel-Medium-Type
      value: IEEE-802
    - type: set
      attribute: Tunnel-Private-Group-Id
      value: "200"
---
# Store accounting records in SQL
apiVersion: radius.operator.io/v1alpha1
kind: RadiusPolicy
metadata:
  name: sql-accounting
  namespace: radius
spec:
  clusterRef: campus-wifi
  stage: accounting
  priority: 100
  actions:
    - type: call
      module: sql
```

---

## ISP Broadband with SQL

An internet service provider authenticating PPPoE/BNG subscribers against a SQL database. High availability with autoscaling and multiple BNG clients across the network.

### Secrets

```bash
kubectl create secret generic subscriber-db-creds \
  --namespace=radius \
  --from-literal=password='pr0dDbP@ss'

kubectl create secret generic bng-shared-secret \
  --namespace=radius \
  --from-literal=secret='BNG-Radius-2026!'
```

### RadiusCluster

```yaml
apiVersion: radius.operator.io/v1alpha1
kind: RadiusCluster
metadata:
  name: subscriber-auth
  namespace: radius
spec:
  image: freeradius/freeradius-server:3.2.3
  replicas: 5
  resources:
    requests:
      cpu: "1"
      memory: 1Gi
    limits:
      cpu: "4"
      memory: 2Gi
  autoscaling:
    enabled: true
    minReplicas: 5
    maxReplicas: 20
    targetCPUUtilizationPercentage: 65
  modules:
    - name: sql
      type: sql
      enabled: true
      sql:
        driver: postgresql
        server: subscriber-db.internal
        port: 5432
        database: radius_subscribers
        credentialsRef:
          name: subscriber-db-creds
          key: password
    - name: redis
      type: redis
      enabled: true
      redis:
        server: redis-cluster.internal
        port: 6379
```

### RadiusClients — BNG Routers

```yaml
apiVersion: radius.operator.io/v1alpha1
kind: RadiusClient
metadata:
  name: bng-region-north
  namespace: radius
spec:
  clusterRef: subscriber-auth
  ip: 172.16.0.0/16
  secretRef:
    name: bng-shared-secret
    key: secret
  nasType: juniper
  metadata:
    region: north
    role: bng
---
apiVersion: radius.operator.io/v1alpha1
kind: RadiusClient
metadata:
  name: bng-region-south
  namespace: radius
spec:
  clusterRef: subscriber-auth
  ip: 172.17.0.0/16
  secretRef:
    name: bng-shared-secret
    key: secret
  nasType: juniper
  metadata:
    region: south
    role: bng
```

### RadiusPolicies

```yaml
# Look up subscriber in SQL during authorization
apiVersion: radius.operator.io/v1alpha1
kind: RadiusPolicy
metadata:
  name: sql-authorize
  namespace: radius
spec:
  clusterRef: subscriber-auth
  stage: authorize
  priority: 100
  actions:
    - type: call
      module: sql
---
# Record accounting to SQL
apiVersion: radius.operator.io/v1alpha1
kind: RadiusPolicy
metadata:
  name: sql-accounting
  namespace: radius
spec:
  clusterRef: subscriber-auth
  stage: accounting
  priority: 100
  actions:
    - type: call
      module: sql
---
# Reject suspended accounts
apiVersion: radius.operator.io/v1alpha1
kind: RadiusPolicy
metadata:
  name: reject-suspended
  namespace: radius
spec:
  clusterRef: subscriber-auth
  stage: authorize
  priority: 50
  match:
    all:
      - attribute: User-Name
        operator: "=~"
        value: "^suspended-"
  actions:
    - type: reject
```

---

## VPN Gateway with REST API

A VPN concentrator that authenticates users against an external identity API over HTTPS. Lightweight — no database, no LDAP.

### Secrets

```bash
kubectl create secret generic vpn-api-token \
  --namespace=radius \
  --from-literal=token='Bearer eyJhbGciOiJSUzI1NiIs...'

kubectl create secret generic vpn-shared-secret \
  --namespace=radius \
  --from-literal=secret='VPN-R@dius-Key'
```

### Resources

```yaml
apiVersion: radius.operator.io/v1alpha1
kind: RadiusCluster
metadata:
  name: vpn-auth
  namespace: radius
spec:
  image: freeradius/freeradius-server:3.2.3
  replicas: 2
  resources:
    requests:
      cpu: 250m
      memory: 256Mi
  modules:
    - name: rest
      type: rest
      enabled: true
      rest:
        server: https://identity-api.internal
        authEndpoint: /v2/radius/authorize
        acctEndpoint: /v2/radius/accounting
        credentialsRef:
          name: vpn-api-token
          key: token
---
apiVersion: radius.operator.io/v1alpha1
kind: RadiusClient
metadata:
  name: vpn-concentrator
  namespace: radius
spec:
  clusterRef: vpn-auth
  ip: 10.100.0.1
  secretRef:
    name: vpn-shared-secret
    key: secret
  nasType: other
  metadata:
    role: vpn-gateway
    vendor: palo-alto
---
apiVersion: radius.operator.io/v1alpha1
kind: RadiusPolicy
metadata:
  name: rest-authorize
  namespace: radius
spec:
  clusterRef: vpn-auth
  stage: authorize
  priority: 100
  actions:
    - type: call
      module: rest
---
apiVersion: radius.operator.io/v1alpha1
kind: RadiusPolicy
metadata:
  name: rest-accounting
  namespace: radius
spec:
  clusterRef: vpn-auth
  stage: accounting
  priority: 100
  actions:
    - type: call
      module: rest
```

---

## Multi-Tenant Isolation

Run separate RADIUS infrastructures for different tenants in the same Kubernetes cluster, using namespace isolation.

```bash
# Create tenant namespaces
kubectl create namespace tenant-acme
kubectl create namespace tenant-globex

# Each tenant gets their own CRD instances
kubectl apply -f acme-cluster.yaml -n tenant-acme
kubectl apply -f globex-cluster.yaml -n tenant-globex
```

Each `RadiusCluster` is fully independent — its own Deployment, Service, ConfigMap, and HPA. Resources in one namespace cannot reference Secrets or clusters in another.

```
tenant-acme/                     tenant-globex/
├── RadiusCluster: acme-radius   ├── RadiusCluster: globex-radius
├── RadiusClient: acme-switch    ├── RadiusClient: globex-ap
├── RadiusPolicy: acme-vlan      ├── RadiusPolicy: globex-guest
├── Secret: acme-db-creds        ├── Secret: globex-db-creds
├── Deployment: acme-radius      ├── Deployment: globex-radius
├── Service: acme-radius         ├── Service: globex-radius
└── ConfigMap: acme-radius-cfg   └── ConfigMap: globex-radius-cfg
```

!!! tip
    Use Kubernetes `ResourceQuotas` and `NetworkPolicies` to further isolate tenants and prevent resource contention.
