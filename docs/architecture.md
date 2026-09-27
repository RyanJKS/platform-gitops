# Architecture

Terraform provisions Azure clusters and owns the Microsoft Argo CD extension.
Each Argo CD root selects a cluster's `argocd/kustomization.yaml`. Kustomize renders
direct Applications and required AppProjects; no ApplicationSet is selected.
Application source code and image builds live in other repositories.

## Configured clusters

| Cluster | Platform selection | Workloads | Argo CD owner |
| --- | --- | --- | --- |
| Azure `uks` `aks-shared` | cert-manager and production issuer | Chaos generator, two replicas | Azure extension |
| Azure `eus2` `aks-shared` | None | Chaos generator, one replica | Azure extension |
| Azure `eus2` `aks-atlas-market` | None | Atlas Market API and worker examples | Azure extension |
| Local `kind-platform` | Argo CD | None | Helm, then GitOps |
| AWS `eks-shared` | None | None | Not selected |

UK South imports `catalog/platform/cert-manager`. One Application combines the
Helm chart and issuer; a Sync hook checks API admission before applying the issuer.
See [HTTPS bootstrap](https.md) for versions, DNS, ordering, and external settings.
Local kind imports `catalog/platform/argocd`; see [local kind](local-kind.md).

Market retains its workload AppProject and selects the shared `atlas-market`
namespace directly in its root. The old namespace bundle, policy, gateway,
monitoring, external-secrets, and cert-manager selections were removed with their
catalog bundles. Its HTTPRoute still requires externally supplied Gateway API CRDs
and a compatible Gateway/controller; it is not a complete ingress deployment.
Workload replicas and images remain unchanged. East US 2's unused Atlas ML files
remain available but are not selected by its current shared root.

## Ownership and deletion

Azure shared roots are Terraform-owned. Market retains its cluster bootstrap root;
local kind uses `bootstrap/local-kind.yaml` outside its cluster tree. Roots never
include themselves in their managed Kustomizations. Helm renders upstream charts
inside child Applications; it does not independently manage an Argo CD release.

Removing a Git selection does not prove that live resources have been removed.
Legacy ApplicationSets can continue managing their Applications until retired.
Review root pruning, owner references, finalizers, and namespace ownership before
migration. See [legacy migration](bootstrap.md#migrate-existing-platform-applications).
No cluster adoption or deletion is performed by repository validation.

## Validation

`scripts/validate.sh` tests direct root selections, renders every Kustomization,
checks root and child paths, and renders selected Helm charts. The old Go
ApplicationSet expander and its catalog-dependent tests were removed. The retained
Go test module checks current Azure and local roots. Cert-manager checks cover
issuer schema, hook ordering, and AppProject permissions.

CI builds TechDocs separately. Offline checks cannot prove cluster health, live
ownership, DNS, certificate issuance, or HTTPRoute readiness.
