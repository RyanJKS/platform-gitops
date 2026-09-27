# Opt-in starter profiles

These profiles preserve the previously repeated cluster configuration. They are
explicit choices, not mandatory policy or production-ready defaults.

- `quota/`: existing workload budget; set the namespace in the cluster overlay.
- `gateway/`: shared GatewayClass and the existing HTTP Gateway listener.
- `workload-namespace/`: Kustomize component applying the existing labels only to
  `atlas-ml` and `atlas-market`. Add a reviewed target for other namespaces.

Cluster Kustomize patches run after these shared definitions. Keep different
permissions, quotas, TLS, listener exposure, and namespace policies local. Review
profile changes for every consumer before merging.
