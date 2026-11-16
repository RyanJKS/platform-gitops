# Observability catalog

`catalog/observability/` groups independent Argo CD Applications in the
`observability` namespace. Select component folders in cluster Kustomizations.
The parent folder installs nothing. See the
[catalog overview](https://github.com/RyanJKS/platform-gitops/blob/HEAD/catalog/observability/README.md).

| Component | Purpose | Shared retention |
| --- | --- | --- |
| kube-prometheus-stack | Scrape metrics, evaluate rules, Grafana, Alertmanager | Prometheus `7d`; Alertmanager `120h` |
| Mimir | Optional remote metrics storage; Kafka ingest; MinIO object storage | Blocks `30d` |
| Loki | Pod logs from Alloy | `7d` |
| Tempo | Application traces from Alloy | `48h` |
| Alloy | Pod log collection and OTLP trace forwarding | No backend retention |

## Select components

- Import required component folders into the cluster's app Kustomization.
- Use project `platform-apps`. Allow the Git repository and each selected Helm repository.
  Mimir and Alloy use `https://grafana.github.io/helm-charts`;
  Loki and Tempo use `https://grafana-community.github.io/helm-charts`;
  kube-prometheus-stack uses `https://prometheus-community.github.io/helm-charts`.
- Allow the `observability` destination and rendered resource kinds in the AppProject.
  Mimir renders namespaced workloads, Services, ConfigMaps, PVCs, Jobs, ServiceAccounts,
  and ServiceMonitors. kube-prometheus-stack also needs cluster resources and CRDs.
- Patch Git values sources from `main` to the published deployment revision.
- Keep a default StorageClass. Review resource capacity and disk growth before sync.
- Prometheus Operator installs at wave `-2`; monitored backends at `-1`; Alloy at `0`.
  Parent Application health checks are required to wait between waves.
- Automatic sync and self-healing are enabled; pruning is disabled.
  Review existing PVCs before changing releases or storage settings.

## Add Mimir

Import `catalog/observability/mimir` through a cluster overlay, following the same
pattern as other catalog Applications. Requires Kubernetes `1.32+` and Prometheus
Operator CRDs. This profile uses one replica per enabled component, one Kafka
broker, and standalone MinIO. No HA. Persistent capacity totals `65Gi` before
other observability components. Size disks for actual ingestion volume.

Before sync, provision Secret `mimir-minio` in namespace `observability` through
the cluster's secret-management workflow. Required keys: `rootUser` and
`rootPassword`. Keep credentials out of Git. Both MinIO and Mimir use this Secret;
the bucket-creation Job also expects this exact Secret name. Changing credentials
requires coordinated rotation and restarting consumers.

Mimir does not replace the Prometheus scraper. Add these values to the cluster's
kube-prometheus-stack Application under `spec.sources[0].helm.valuesObject` when
selecting Mimir:

```yaml
prometheus:
  prometheusSpec:
    remoteWrite:
      - url: http://mimir-gateway.observability.svc.cluster.local/api/v1/push
grafana:
  additionalDataSources:
    - name: Mimir
      uid: mimir
      type: prometheus
      access: proxy
      url: http://mimir-gateway.observability.svc.cluster.local/prometheus
      isDefault: false
```

Helm replaces lists. Include existing Loki and Tempo entries when overriding
`grafana.additionalDataSources`. Prometheus stays the default data source.
Keep rules and alerts in kube-prometheus-stack; Mimir ruler and Alertmanager are disabled.

The gateway uses ClusterIP port `80`. Multi-tenancy is disabled; no tenant header
or API authentication is required. Restrict network access in shared clusters.
No public ingress is created. For production, replace bundled Kafka and MinIO
with production services and configure replicas, authentication, and capacity.
See [upstream production guidance](https://grafana.com/docs/helm-charts/mimir-distributed/latest/run-production-environment-with-helm/).

## Verify

Render the catalog Application and pinned Helm chart before selecting it:

```sh
kubectl kustomize catalog/observability/mimir
helm template mimir mimir-distributed --repo https://grafana.github.io/helm-charts \
  --version 6.2.2 --namespace observability --kube-version 1.32.0 \
  --skip-tests -f catalog/observability/mimir/values.yaml
```

After deployment:

```sh
kubectl -n observability get pods,pvc,svc,servicemonitors
kubectl -n observability port-forward svc/mimir-gateway 8080:80
```

In another terminal, query stored metrics after Prometheus remote write starts:

```sh
curl --fail --get http://localhost:8080/prometheus/api/v1/query --data-urlencode 'query=up'
```

Expect successful queries with scraped series. Check Prometheus remote write
errors and pending samples if results stay empty. Gateway `/ready` checks the
proxy only; it does not prove ingestion or storage health.
If bucket creation fails, inspect the `mimir-make-minio-buckets-5.4.0` Job logs.
Its Sync hook is removed after success and recreated before the next sync.

Grafana remains in kube-prometheus-stack. Discover its Service with
`kubectl -n observability get svc`, port-forward that Service to port `3000`, and
use its chart-generated admin Secret. Verify Mimir through Explore with the
Mimir data source. Send known pod logs and instrumented application traces to
verify Loki and Tempo separately. See [debugging](debugging.md) for Argo CD checks.
