# Cluster bootstrap

Apply exactly one root to the matching cluster after installing Argo CD.
Each root targets the in-cluster Kubernetes API; it does not register or select a remote cluster.

| Root | Cluster | Workloads |
| --- | --- | --- |
| `azure-dev-uksouth-atlas-shared.yaml` | Azure dev `aks-shared` | Atlas ML inference |
| `azure-dev-uksouth-atlas-market.yaml` | Azure dev `aks-atlas-market` | Atlas Market API and worker |
| `local-kind.yaml` | Local `kind-platform` | None |

Follow the complete [bootstrap procedure](../docs/bootstrap.md), including manual sync order.
Do not apply the whole roots directory. Roots intentionally omit automated sync and deletion finalizers.
