# Architecture

The Azure shared cluster variants (`uks` and `eus2`) select only `chaos-generator`,
using automatic sync and pruning under Terraform-owned `all-apps`. The `eus2` child
reads the catalog base at one replica. The `uks` child reads a cluster overlay that
imports that base and sets two replicas. No platform components are selected.
The `eus2` legacy root and platform files are inactive. See the
[bootstrap contract and pruning review](bootstrap.md#azure-shared-chaos-generator-only).
The platform architecture below describes the other cluster examples.

## Repository and rendering boundaries

Terraform provisions clusters. Each cluster's own Argo CD instance reconciles
platform components and business applications from this public repository.
Application code and image builds live elsewhere.

Each cluster owns a small `bootstrap/root.yaml`. Its full repository-relative
source path selects the cluster's `argocd/kustomization.yaml`; that entry point
never includes the root. Azure and AWS retain their cloud/account/environment/
region/spoke hierarchy. Local kind retains its shorter hierarchy.

Kustomize composes shared AppProjects, one platform ApplicationSet, and unique
workload Applications. The ApplicationSet uses an explicit list plus Git-file
matrix: the list chooses component names, the Git generator reads only those
catalog component files, and its Go template constructs Applications. The cluster
entry point supplies one `clusterPath`; no folder discovery or implicit path
parameters are involved. Helm is used only by child Applications to render their
upstream charts. Kustomize renders resource overlays.

```text
catalog/platform/
  applicationsets/platform/    reusable ApplicationSet template
  components/                 chart versions and component input data
  projects/platform/          shared platform permission base
  profiles/starter/            opt-in quota, namespace policy, HTTP Gateway
  <component>/values.yaml      common Helm defaults
catalog/applications/<app>/
  project/                    matching workload permissions
  namespace/                  namespace identity
  <component>/base/           reusable workload resources
clusters/.../<cluster>/
  bootstrap/root.yaml         initial root, not self-managed
  argocd/kustomization.yaml   explicit selections and cluster identity
  argocd/projects/            shared project selection and permission patches
  argocd/applications/        unique workload Applications, when present
  platform/                  shared resource selections and value differences
  applications/              workload images, routes, and runtime overrides
```

See [configuration boundaries](configuration-boundaries.md) for precedence and
[adding a cluster](adding-a-cluster.md) for complete examples.

## Configured clusters

| Cluster | Generated platform Applications | Workloads | Argo CD owner |
| --- | --- | --- | --- |
| Azure `eus2` `aks-shared` | None | Chaos generator | Azure extension |
| Azure `uks` `aks-shared` | None | Chaos generator | Azure extension |
| Azure `eus2` `aks-atlas-market` | Seven | Atlas Market API and worker | Azure extension |
| Local `kind-platform` | Eight, including Argo CD | None | Helm, then GitOps |
| AWS `eks-shared` | None; reserved | None | Not selected |

Legacy Azure root names retain `uksouth` for identity compatibility and use `eus2`
paths. Terraform-owned `all-apps` can select the `uks` or `eus2` shared variant. This does not move cloud resources. Azure roots do not deploy
Argo CD or overwrite extension-owned ConfigMaps. AWS remains unconfigured.

## Reconciliation and deletion

The other cluster examples retain manual sync without workload pruning. Generated Applications have ApplicationSet controller
owner references after adoption, but no resource-deletion finalizer because
`preserveResourcesOnDeletion` is true. `applicationsSync: create-update` requests
no automatic Application deletion on selection removal; the controller must honor
that per-set policy. See the required [migration procedure](bootstrap.md#migrate-existing-platform-applications).

Root sync creates projects, the ApplicationSet, and ordinary Applications. The
ApplicationSet controller subsequently creates or adopts platform Applications.
Selected catalog component changes update generated Application specs on ApplicationSet
reconciliation; this is the intentional control-plane behavior change from ordinary
root-managed Applications. Workloads still require manual sync. Cluster selection
and template changes require root sync. Sync waves do not make child Applications sync automatically. Follow the explicit
order in [bootstrap](bootstrap.md).

## Validation and limits

CI builds Kustomize entry points, expands this repository's list/Git-file matrix
against the checkout using Go templates and Sprig, verifies selection scope, and
renders pinned Helm charts with effective values. The offline expander supports
only this declared generator shape and fails on unsupported input. It does not
contact Argo CD or prove live adoption, owner references, controller policy flags,
or reconciliation against the remote Git revision. Argo CD and Gateway API schemas
remain excluded from kubeconform. TechDocs builds separately in strict mode.

Other example images and DNS names remain placeholders, with zero workload replicas.
Chaos generator uses its upstream image, with one replica in `eus2` and two in
`uks`; runtime is unverified.
TLS, cloud identity, persistence, network isolation, and recovery require cluster
implementation and runtime validation before production use.
