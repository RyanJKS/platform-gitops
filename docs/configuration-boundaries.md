# Configuration boundaries

Keep reusable manifests and Helm defaults in `catalog/`. Keep cluster selections
and workload overrides under `clusters/`. A directory name does not grant access
or implicitly select a deployment.

- `catalog/platform/cert-manager/` contains the direct Application, AppProject,
  Helm values, and issuer used by UK South.
- `catalog/platform/argocd/` contains the direct Application and Helm values used
  by local kind. Azure never selects this installation.
- `catalog/applications/` contains reusable workloads, namespaces, and AppProjects.
- Cluster `argocd/kustomization.yaml` files explicitly select direct Applications,
  AppProjects, and required namespaces.
- Cluster `applications/` directories contain workload image, replica, and routing
  overrides. Local kind currently has no workload selection.

The deleted platform ApplicationSet, component inputs, and starter profiles are
no longer configuration interfaces. Add only resources needed by a selected
cluster; do not recreate speculative platform bundles.

## Installation ownership

Terraform owns Azure shared roots and the Argo CD extension, including its Ingress
and ConfigMaps. Configure those resources through the extension's supported
Terraform settings. Cert-manager owns generated ACME and certificate Secrets.

Local kind retains its Helm-to-GitOps handover and a root stored at
`bootstrap/local-kind.yaml`. Market retains its bootstrap root and workload
AppProject. Its namespace is selected directly by the root, not by the deleted
platform namespace Application. Review live ownership before adopting that namespace.

## Values and permissions

A chart's Application pins its version and lists shared Helm values. Later value
files override earlier ones. UK South adds Argo CD startup-hook values after the
shared CRD setting. Cluster-specific values belong under that cluster only when
there is a real difference; point the Application at those files explicitly.

Keep workload AppProjects narrow. Include namespaces and permissions required by
new workloads rather than relying on the former platform namespace bundle.
Shared catalog changes affect every selected consumer. Review those consumers
before changing shared policies or values, and never commit generated secrets.
