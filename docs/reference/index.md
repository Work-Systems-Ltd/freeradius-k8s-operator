---
title: CRD Reference
nav_order: 4
has_children: true
---

# CRD Reference

Complete specification for the three custom resources that define your RADIUS infrastructure.

All resources belong to the API group `radius.operator.io/v1alpha1` and are namespace-scoped.

| Kind | Short Name | Description |
|:-----|:-----------|:------------|
| [RadiusCluster](radiuscluster/) | `rc` | Defines a FreeRADIUS deployment — image, replicas, modules, TLS, autoscaling |
| [RadiusClient](radiusclient/) | `rcl` | Registers a network device (NAS) authorized to send RADIUS requests |
| [RadiusPolicy](radiuspolicy/) | `rp` | Declares authentication/authorization logic as match conditions and actions |
