# TLS Configuration

Secure RADIUS traffic with TLS (RadSec).

---

## Overview

Standard RADIUS uses UDP with a shared secret for packet-level authentication. For environments that require transport-layer encryption, the operator supports TLS-based RADIUS (RadSec, RFC 6614), which wraps RADIUS traffic in a TLS tunnel over TCP port 2083.

## Create the TLS Secret

Store your certificate and private key in a Kubernetes Secret:

```bash
kubectl create secret tls radius-tls \
  --namespace=radius \
  --cert=server.crt \
  --key=server.key
```

Or from separate files:

```bash
kubectl create secret generic radius-tls \
  --namespace=radius \
  --from-file=tls.crt=server.crt \
  --from-file=tls.key=server.key \
  --from-file=ca.crt=ca.crt
```

## Enable TLS

```yaml
apiVersion: radius.operator.io/v1alpha1
kind: RadiusCluster
metadata:
  name: secure-radius
  namespace: radius
spec:
  image: freeradius/freeradius-server:3.2.3
  replicas: 2
  tls:
    enabled: true
    secretRef:
      name: radius-tls
      key: tls.crt
```

When TLS is enabled, the operator:

1. Mounts the TLS Secret as a read-only volume
2. Configures FreeRADIUS to listen on the RadSec port (2083/TCP)
3. References the certificate and key paths in `radiusd.conf`

## Using cert-manager

For automated certificate lifecycle management, use [cert-manager](https://cert-manager.io/) to provision and rotate certificates:

```yaml
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: radius-tls
  namespace: radius
spec:
  secretName: radius-tls
  issuerRef:
    name: internal-ca
    kind: ClusterIssuer
  commonName: radius.internal
  dnsNames:
    - radius.internal
    - "*.radius.svc.cluster.local"
  duration: 8760h    # 1 year
  renewBefore: 720h  # 30 days
```

cert-manager updates the Secret in place when it rotates the certificate. The operator detects the change and triggers a rolling update to pick up the new certificate.

## EAP-TLS

For EAP-TLS (certificate-based 802.1X authentication), configure the TLS certificate in the EAP module rather than the cluster-level TLS setting:

```yaml
modules:
  - name: eap
    type: eap
    enabled: true
    eap:
      defaultMethod: tls
      tls:
        certRef:
          name: eap-tls-cert
          key: tls.crt
        keyRef:
          name: eap-tls-cert
          key: tls.key
```

Cluster-level TLS secures the RADIUS transport (RadSec). EAP-TLS secures the authentication method. These are independent and can be used separately or together.
