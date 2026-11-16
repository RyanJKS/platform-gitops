# Tempo

- Chart `3.2.1`; Tempo `3.1.1`. Single binary; one replica; `10Gi` local PVC; `48h` retention.
- OTLP gRPC `4317`, OTLP HTTP `4318`, query API `3200`. Chart retains default Jaeger receivers.
- ServiceMonitor enabled. Requires Prometheus Operator CRDs before sync.
- Applications send traces through Alloy. No automatic application instrumentation.

```sh
kubectl kustomize catalog/observability/tempo
```

See [observability catalog](../README.md) and [operations](../../../docs/observability.md).
