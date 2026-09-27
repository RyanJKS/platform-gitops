# Local kind

The local cluster now uses the same two-directory layout as UK South:

```text
clusters/local/kind/kind-platform/
  argocd/kustomization.yaml
  applications/README.md
```

`argocd/` selects one direct `platform-argocd` Application from
`catalog/platform/argocd`. It retains chart `argo-cd` version `7.8.13`, release name
`argocd`, namespace `argocd`, and the shared Helm values. It uses the existing
`default` AppProject and manual sync without pruning or a deletion finalizer.
No workloads are enabled by default. No Azure HTTP-01 issuer is selected locally.

The former platform ApplicationSet, platform/atlas-ml projects, namespace bundle,
cert-manager, external-secrets, monitoring, gateway, and policy selections are
removed from kind. Market and UK South selections are unchanged. The obsolete
`components/argocd.yaml` generator input is removed; the direct Application now
owns that chart pin. Shared catalog resources still used by market remain.

## Bootstrap a new cluster

Run from the repository root after the manifests are available on `main`.
The initial Argo CD installation is external to its own GitOps reconciliation,
as with the extension-owned installation on AKS. These commands apply only to kind:

```sh
kind create cluster --name kind-platform
CONTEXT=kind-kind-platform
helm repo add argo https://argoproj.github.io/argo-helm
helm repo update argo
helm upgrade --install argocd argo/argo-cd --version 7.8.13 \
  --kube-context "$CONTEXT" --namespace argocd --create-namespace \
  -f catalog/platform/argocd/values.yaml --wait
kubectl --context "$CONTEXT" apply -f bootstrap/local-kind.yaml
kubectl --context "$CONTEXT" -n argocd port-forward service/argocd-server 8080:443
```

The root keeps the name `local-kind` and the same source path. It is stored outside
the cluster directory and outside its own rendered resources. In another terminal,
log in to Argo CD at `localhost:8080` using the installation's credentials and
self-signed-certificate handling. Sync `local-kind`, then `platform-argocd`.
Check both Applications report Synced and Healthy. After handover, manage Argo CD
through its Git source rather than subsequent Helm upgrades or uninstalls.

The default AppProject must allow the Git and Argo Helm repositories, the local
`argocd` destination, and the chart's namespaced and cluster-scoped resources.
Confirm its permissions if the installation overrides the standard default project.

## Migrate an existing kind installation

Changing Git selections does not uninstall existing platform resources. Keep
pruning disabled until ownership is reviewed. The old ApplicationSet can otherwise
continue managing Applications, including `platform-argocd`, and conflict with the
new direct Application.

Before syncing the new root, save the old ApplicationSet and Applications, inspect
owner references and finalizers, and verify that the root has no automatic pruning.
Retire the old `platform` ApplicationSet with orphan propagation after that review,
preserving its generated Applications and their workloads. Remove only obsolete
root-tracking metadata from `platform-argocd` if it prevents adoption, then sync the
new root without pruning. Verify `platform-argocd` retains its UID and has the new
`default` project, source, and root tracking; no ApplicationSet may still control it.

Review the seven other former child Applications separately. Removing an Application
with a resource-deletion finalizer can uninstall workloads; namespace deletion can
also remove unrelated resources. Retire unwanted resources deliberately after
checking their consumers. Do not use root pruning as a blanket uninstall. Remove
old AppProjects only after no Applications reference them. This repository change
does not perform any live adoption, deletion, or uninstall.

## Add a local workload

Put its overlay under `applications/<name>/`, add its child Application under
`argocd/`, and list that manifest in `argocd/kustomization.yaml`. Supply its namespace
and permissions as part of the workload configuration; the former atlas-ml
namespace and AppProject are no longer selected. Follow
[application onboarding](onboarding-an-application.md).

Render the entry point with:

```sh
kubectl kustomize clusters/local/kind/kind-platform/argocd
```

It must contain only the `platform-argocd` Application until a workload is added.
