# kube-prometheus-stack

- Prometheus Operator, Prometheus, Alertmanager, Grafana.
- Chart: `kube-prometheus-stack` `92.7.0`; destination: `observability`.
- Argo CD imports this catalog base; `CreateNamespace=true` creates destination namespace.
- AppProject must allow `ghcr.io/prometheus-community/charts`, destination namespace, and rendered resource types.
- Single OCI Helm source uses upstream defaults.
- Cluster overrides belong in `spec.source.helm.valuesObject`.

Render from repository root:

```sh
kubectl kustomize catalog/platform/kube-prometheus-stack
```

[Upstream chart](https://github.com/prometheus-community/helm-charts/tree/main/charts/kube-prometheus-stack): values and operational guidance.
