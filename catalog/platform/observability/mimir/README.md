# Mimir

- Official `mimir-distributed` chart `6.2.2`; Mimir `3.2.3`. Kubernetes `1.32+`.
- Development profile. One replica per enabled Mimir component; no HA or zone replication.
- Ingest storage enabled. Bundled single-broker Kafka (`10Gi`) and standalone MinIO (`30Gi`).
- Ingester `10Gi`, store-gateway `5Gi`, compactor `10Gi` PVCs. Default StorageClass required.
- Blocks retained for `30d`. Size object storage for ingestion volume; retention is not a disk quota.
- Requires existing `observability/mimir-minio` Secret with `rootUser` and `rootPassword` keys.
  Mimir reads credentials through environment variables; MinIO and bucket Job use the same Secret.
- Bucket creation runs as an Argo CD Sync hook; recreated before sync and removed after success.
- ServiceMonitor enabled. Requires Prometheus Operator CRDs before sync.
- ClusterIP gateway `mimir-gateway:80`; remote write `/api/v1/push`; Prometheus queries `/prometheus`.
- Multi-tenancy disabled. No API authentication or public ingress. Restrict access in shared clusters.
- Ruler and Mimir Alertmanager disabled. Keep rules and alerts in kube-prometheus-stack.
- Opt-in only. Select the Application and configure Prometheus remote write and Grafana in cluster overlays.

```sh
kubectl kustomize catalog/observability/mimir
helm template mimir mimir-distributed --repo https://grafana.github.io/helm-charts \
  --version 6.2.2 --namespace observability --kube-version 1.32.0 \
  --skip-tests -f catalog/observability/mimir/values.yaml
```

For production, use external object storage and Kafka, replicated components,
authentication, and capacity planning. See [upstream production guidance](https://grafana.com/docs/helm-charts/mimir-distributed/latest/run-production-environment-with-helm/).

See [observability catalog](../README.md) and [operations](../../../docs/observability.md).
