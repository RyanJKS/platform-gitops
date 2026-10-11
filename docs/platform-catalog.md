# Select platform configuration

`catalog/platform/<component>/` owns each reusable Application, chart pin, shared
values, provider files, and controller configuration. Cluster files select the
components and supply identities, DNS settings, permissions, and sync policy.
Catalog manifests must not reference paths under `clusters/`. Cluster overlays
import catalog bases and set their own child config paths and Git revisions.

## Application patches and Helm values

Cluster `application-patches.yaml` combines Application changes with cluster Helm
configuration under `spec.sources[0].helm.valuesObject`. No separate cluster
`values.yaml` is needed for Argo CD, External DNS, or AGIC. The local app
Kustomization imports its catalog Application and applies this single patch file.

The AKS AGIC sandbox's Argo CD patch contains a complete `valuesObject`, combining
defaults from `catalog/platform/argocd/application.yaml`, Azure settings from
`catalog/platform/argocd/configs/azure/values.yaml`, and cluster overrides, in that
order. This object replaces the catalog object; Kustomize does not merge the
maps. Keep the copied defaults and Azure settings aligned when their catalog
sources change. The Application uses one Helm source, no `valueFiles`, and no
`$values` Git source. Exclude Argo CD from the parent's Git revision patch.

For components that use `valueFiles`, keep shared and provider values in the
catalog and select the files in the Application. Helm's inline `valuesObject`
takes precedence over those files.

External DNS's catalog Git source is values-only: it has `ref: values` and no
cluster path. The cluster Application patch appends the provider values file and
sets `spec.sources[1].path` to that cluster's controller config overlay. Cert-manager
uses the same pattern for its issuer config source. Selecting a provider or
cluster never changes the catalog. `$values/` paths are repository-root paths.
Chart versions, sync waves, and sync policy remain catalog defaults.

## Catalog sync defaults

Platform and sandbox workload AppProjects use wave `-4`. Catalog Applications
define their child sync waves: AGIC `-3`, External DNS `-2`,
cert-manager `-1`, and chaos-generator `0`. They also enable automated sync and
self-healing with pruning disabled. Cluster patches retain only deployment
choices such as DNS provider, Git revision, and issuer or workload configuration.
A cluster with different sync requirements must override the catalog defaults
explicitly.
Opt-in Kyverno and Falco use wave `-1` so their controllers also precede workloads.
Lower waves apply first. Workloads without an annotation use wave `0`.
Cert-manager chart hooks and issuer waves are evaluated
inside its child Application and keep their existing values.

Waves order child Applications when their parent syncs. The shared Argo CD Helm values provide the Application health
check that makes the parent wait for each child to report `Synced` and `Healthy`.
Other Argo CD installations need equivalent Application health checks to wait
between child waves. Existing children still reconcile independently.

## Provider selection

Provider selection is a cluster-specific patch. The AGIC sandbox uses:

```yaml
- op: add
  path: /spec/sources/0/helm/valueFiles/-
  value: $values/catalog/platform/external-dns/azure-values.yaml
```

Use `aws-values.yaml` for Route 53 or `gcp-values.yaml` for Google Cloud DNS.
This appends the provider file, so provider defaults stay before
cluster overrides. Helm merges maps but replaces lists; overrides of `extraArgs`
must retain required provider arguments. `$values/` paths are relative to the Git
repository root, regardless of the Git source's `path`.

Configure cloud authentication, DNS filters, and a unique TXT owner ID in the
cluster Application patch. Keep cloud identities and role assignments with the
infrastructure owner. Align root, values, and configuration Git revisions.

## Controller configuration

The root selects Applications and projects. Child Applications deploy charts and
controller resources. Keep ClusterIssuers and controller ConfigMaps in the child
configuration directory.

For Azure DNS, the cluster configuration imports
`catalog/platform/external-dns/configs` and patches its tenant, subscription, and
resource group. AWS and GCP clusters must provide their own configuration
Kustomization, which may select no resources; do not import the Azure ConfigMap.

Cert-manager has consumers with different issuer layouts. Its cluster-specific
Git source path remains a patch because it selects that cluster's issuer
configuration. Keep issuer names, ACME endpoints, email, and ingress class in
cluster patches. The chart and issuer remain in one Application to preserve
cert-manager's API readiness gate.

Argo CD's catalog `application.yaml` defines the intended self-management child
and runtime chart version. Shared runtime values stay in `values.yaml`, Azure
settings in `azure-values.yaml`, and deployment overrides in the cluster's
`platform/argocd/application-patches.yaml` under `helm.valuesObject`. Argo CD
consumes the selected values files and inline overrides after handoff.
Terraform installs its own pinned minimal chart with chart defaults.

Catalog `root_project.yaml` is a reusable GitOps AppProject base included in the
Argo CD catalog Kustomization. The cluster root creates that `bootstrap` project
before its self-management child. The infrastructure Application unit defines
`all-apps` inline in the built-in `default` project and selects the cluster root
on `dev/sandbox`. Terraform does not create a separate AppProject.

The retained `catalog/platform/argocd/bootstrap/` files are optional reusable
templates excluded from the active Kustomization. The root Application template
uses `REPLACE_PLACEHOLDER` for its path. Set deployment inputs before using that
alternative bootstrap procedure; the active Terraform root is defined inline.

The cluster root renders only projects and child Applications. Parent
Kustomizations select local app folders: `platform/` selects `argocd`,
`cert-manager`, `external-dns`, and `ingress-agic`; `workloads/` selects
`chaos-generator`. Each local app Kustomization imports its catalog Application
and applies its own `application-patches.yaml` where required. The platform
Kustomization sets all values/config Git sources to `dev/sandbox`.

Cert-manager, External DNS, and chaos-generator child Git sources point at their
app's separate `configs/` path. That Kustomization imports catalog controller
configuration or workload resources and applies resource patches stored beside
it. Application patches remain next to the app-level Kustomization. These are
two render boundaries: the root creates Applications; children create runtime
resources. Keep the child source separate to prevent rendering its own
Application. Argo CD and AGIC use Helm with a values-only Git source, so they do
not need a cluster config Kustomization.

Argo CD values merge in shared, Azure, then cluster order. Its cluster patch keeps
automatic sync disabled until ownership handoff. Validate root selection and
child resources before syncing. Follow [bootstrap ownership handoff](bootstrap.md).

## Editable markers and verification

Use `REPLACE_PLACEHOLDER` for every editable marker. The YAML field, shell variable,
or surrounding instructions identify the required value. The provider patch appends its catalog values file. Fill remaining deployment inputs with their actual values.
Do not replace all
markers with one value: each field needs its own configuration. Unselected
catalog templates may retain markers; render selected resources and inspect
remaining markers before deployment.

```sh
kubectl kustomize clusters/azure/DEV-JKS/dev/uks/sandbox/aks-agic
rg -n 'REPLACE_PLACEHOLDER' catalog clusters
```

Rendering checks composition and paths. It does not prove cloud access, DNS, or
controller readiness. See [bootstrap](bootstrap.md) and [debugging](debugging.md)
for deployment and runtime checks.
