# Loki

- Chart `21.0.0`; Loki `3.7.9`. Monolithic mode; one replica; `20Gi` filesystem PVC.
- TSDB schema `v13`; `7d` retention with compactor deletion enabled.
- Direct ClusterIP service `loki:3100`; no gateway, MinIO, distributed targets, or cache Pods.
- ServiceMonitor enabled. Requires Prometheus Operator CRDs before sync.
- Schema start date applies to new storage. Preserve schema history when migrating existing Loki data.

```sh
kubectl kustomize catalog/observability/loki
```

See [observability catalog](../README.md) and [operations](../../../docs/observability.md).
