# UK South AGIC sandbox

This cluster uses Kustomize to bootstrap Argo CD and the `all-apps` root
Application in the `bootstrap` project. The root creates `platform-apps` for
controllers and `apps` for workloads. It selects `cert-manager`, `external-dns`,
and `ingress-agic` under `platform-apps`, plus `chaos-generator` under `apps`.

`bootstrap/kustomization.yaml` imports the shared installation base and applies
cluster-specific Argo CD patches. `bootstrap/seed.yaml` seeds the root after CRD
and controller readiness. Installation and seeding use separate commands; all
bootstrap files live directly in `bootstrap/`. `argocd/` defines both shared projects and selects
catalog-owned Applications, then applies cluster patches, Helm values, and
workload overlays. Its root Kustomization selects both project files, `platform/`,
and `applications/`.

All rendered Applications read Git configuration from `dev/sandbox`. Child
Applications start with manual sync while Azure and DNS placeholders are configured.

See [the AKS AGIC guide](../../../../../../../docs/aks-agic.md) for prerequisites,
required values, bootstrap commands, sync order, access controls, and validation.
