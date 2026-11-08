# kube-prometheus-stack

- Prometheus Operator, Prometheus, Grafana, Alertmanager, node-exporter, kube-state-metrics.
- Shared values: PVCs, `7d` metrics retention, Loki/Tempo data sources, ServiceMonitor and PodMonitor discovery.
- Monitors across all namespaces accepted. Restrict selectors in clusters with untrusted tenants.
- `ServerSideApply=true` handles large Prometheus Operator CRDs.
- Moved from `catalog/platform/kube-prometheus-stack/`; Application and Helm release names stay unchanged.
- Existing deployments need PVC and Helm-source migration review before sync.

Render from repository root:

```sh
kubectl kustomize catalog/observability/kube-prometheus-stack
```

See [observability catalog](../README.md) and [operations](../../../docs/observability.md).
