# Alloy

- Chart `1.15.0`; Alloy `v1.21.3`. One DaemonSet Pod per eligible node.
- Node-filtered discovery prevents duplicate collection across Alloy Pods.
- Kubernetes API reads pod logs. No privileged container or host filesystem mounts.
- Labels: `namespace`, `pod`, `container`, `node`, `app`, `job`. `app` uses `app.kubernetes.io/name` when present.
- OTLP receives traces on `4317`/`4318`; memory limiter and batch processor forward traces to Tempo.
- RBAC reads only Pods, pod logs, and namespaces. ServiceMonitor exposes collector metrics.
- Collection state and trace queues are ephemeral. Restarts can replay logs or lose queued telemetry.

```sh
kubectl kustomize catalog/observability/alloy
```

See [observability catalog](../README.md) and [operations](../../../docs/observability.md).
