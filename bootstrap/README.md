# Cluster bootstrap

Shared examples and instructions live here. Each configured cluster owns its initial
`bootstrap/root.yaml`; reusable definitions stay in `catalog/platform/` and
`catalog/applications/`.

Follow the [bootstrap procedure](../docs/bootstrap.md) for exact paths, installation
methods, context selection, and sync order. Apply only the chosen cluster's root.
Public repositories require no repository credential Secret.

Local kind uses `bootstrap/local-kind.yaml` at repository level. Its cluster tree
contains only `argocd/` and `applications/`. Follow the
[local kind guide](../docs/local-kind.md); the root is not included in its own
Kustomization.
