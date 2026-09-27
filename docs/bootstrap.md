# Bootstrap

## Azure shared: chaos-generator only

For the `uks` and `eus2` variants of `spoke-atlas/aks-shared`, Terraform installs
Argo CD through the Azure extension and creates the root Application `all-apps`
in namespace `argocd`. Do not apply the inactive legacy `bootstrap/root.yaml`,
install another Argo CD instance, or create another root owner.

Configure Terraform's root with this exact source and enable automated sync:

```yaml
spec:
  source:
    repoURL: https://github.com/RyanJKS/platform-gitops.git
    targetRevision: feature/repo-setup
    path: clusters/azure/DEV-JKS/dev/uks/spoke-atlas/aks-shared/argocd
  destination:
    server: https://kubernetes.default.svc
    namespace: argocd
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
```

The source above selects the new `uks` variant. The existing `eus2` variant remains
available at `clusters/azure/DEV-JKS/dev/eus2/spoke-atlas/aks-shared/argocd`.
Choose exactly one source path for each cluster's Terraform-owned `all-apps` root;
do not create both variants' identically named child Applications in one cluster.
Both variants use the same catalog base; `uks` adds a replica override. No bootstrap root manifest is
provided for `uks`: Terraform remains the only root owner. Terraform configuration lives
outside this repository and is not changed here. The repository is public;
no repository credential Secret is required. The existing `default` AppProject
must allow this repository, the in-cluster destination, and workload resources.

Commit and push these changes to `feature/repo-setup` before Argo CD can read them.
Terraform creates the root; Argo CD asynchronously creates the `chaos-generator`
child Application and then its workload. Argo CD shows separate `all-apps` and
`chaos-generator` entries. The child uses automatic sync, pruning, self-healing,
and `CreateNamespace=true`, targeting namespace `chaos-generator`.

The `eus2` child points directly at `catalog/applications/chaos-generator`. The `uks`
child points at its cluster overlay, described below. The catalog Deployment
and Service preserve the functional settings from the upstream
[intro example](https://github.com/RyanJKS/chaos-generator/tree/main/infrastructure/k8s/examples/intro),
inspected on 2026-09-27: one replica, image `ryanjks/chaos-generator:v1`, and TCP port
8501. Namespace is unset in both manifests so Argo CD supplies it. The base excludes
NGINX Ingress and the commented probes targeting port 3000. No platform component,
ApplicationSet, custom AppProject, or ingress controller is selected.
Image pullability and application runtime have not been verified.

### Override replicas and image tag for uks

The `uks` child reads
`clusters/azure/DEV-JKS/dev/uks/spoke-atlas/aks-shared/applications/chaos-generator`.
Its `kustomization.yaml` imports the catalog base and overrides the Deployment:

```yaml
replicas:
  - name: chaos-generator
    count: 2
images:
  - name: ryanjks/chaos-generator
    newTag: v1
```

Edit `count` in that overlay to change the desired replica count. Edit `newTag`
to select an image version; it currently retains the upstream `v1` tag. The shared base
and the `eus2` variant remain at one replica. Commit and push the overlay change
to `feature/repo-setup`; the child's automatic sync applies it. Manual scaling can
be reverted by self-healing. Terraform's root still watches the `uks/…/argocd`
entry point; only the child source points at the workload overlay.

### Review pruning before applying

The new `uks` entry point has no previous Git selections. If an existing root is
repointed to it, review the resources that root currently tracks before pruning.
The previous `eus2` entry point rendered AppProjects `atlas-ml` and `platform`, Application
`atlas-ml-inference`, and ApplicationSet `platform`. Root pruning can delete these
objects if the root previously tracked them. Deleting the ApplicationSet can also
cause Kubernetes garbage collection to delete its generated Applications:
`platform-cert-manager`, `platform-external-secrets`, `platform-monitoring`,
`platform-gateway-controller`, `platform-namespaces`, `platform-policy`, and
`platform-gateway-resources`.

The old ApplicationSet specifies `applicationsSync: create-update` and
`preserveResourcesOnDeletion: true`. The former does not prevent garbage collection
when the ApplicationSet itself is deleted. The latter requests that generated
Applications have no resource-deletion finalizer. Inspect live ownership and finalizers
before applying: deleting an Application with a resource-deletion finalizer can delete
its workloads, including shared namespaces or controllers and resources depending on
them. Without that finalizer, workloads normally remain unmanaged. AppProject deletion
can be delayed by remaining Applications that reference it. A newly created `all-apps`
root does not automatically prune objects tracked by another root; resolve existing
ownership before enabling reconciliation. The unrelated files remain in Git.
No live resources were inspected or deleted for this change.

### Verify and access

Render locally from the repository root:

```sh
kubectl kustomize clusters/azure/DEV-JKS/dev/uks/spoke-atlas/aks-shared/argocd
kubectl kustomize clusters/azure/DEV-JKS/dev/eus2/spoke-atlas/aks-shared/argocd
kubectl kustomize clusters/azure/DEV-JKS/dev/uks/spoke-atlas/aks-shared/applications/chaos-generator
kubectl kustomize catalog/applications/chaos-generator
```

After the changes reach Git and Terraform creates the root, verify the selected
cluster context, then run:

```sh
kubectl get applications -n argocd
kubectl get deployment,pods,service -n chaos-generator
kubectl port-forward -n chaos-generator service/chaos-generator 8501:8501
```

Open `http://localhost:8501`. If the child is missing, check the root's source path,
revision, and sync status. If Pods are not ready, inspect Pod events and image-pull
errors. Do not trigger chaos operations as part of bootstrap verification.

## Other cluster examples

The remaining instructions apply to the market and local kind examples, not the
Terraform-owned `all-apps` setup above.

## Prerequisites

Start with an existing Kubernetes cluster, its explicit kubeconfig context, kubectl, and the
Argo CD CLI. Helm is required only for Helm-based installations. Kubernetes 1.32 is the rendering baseline used by validation, not a tested compatibility
promise for every managed cluster version. Confirm each pinned chart supports your cluster version.
Allow outbound chart and Git access. Configure Argo CD repository credentials through a secure
channel if this repository is private. Never commit credentials.

Merge the configuration to `main` before bootstrap, or consistently change root and child revisions
to a review branch. All repository sources use `https://github.com/RyanJKS/platform-gitops.git`;
update every occurrence if using a fork.

Run commands from the repository root.

## Choose a cluster and installation method

Terraform provisions AKS and installs Argo CD through the Azure extension. On the
Azure dev clusters, verify that extension-managed Argo CD is ready; do not install
another Helm release or sync `platform-argocd`. These clusters omit the `argocd` component
and do not reconcile extension-owned ConfigMaps.

The repository is public, so no repository credential Secret is required.
Bootstrap must run against the intended Kubernetes context. Each root targets the
in-cluster API; its manifest does not select or register a remote cluster.

Before bootstrap, verify that the installed ApplicationSet controller supports Go
templates, `templatePatch`, matrix generators, and `preserveResourcesOnDeletion`.
It must honor `spec.syncPolicy.applicationsSync: create-update`: check its effective
policy and whether per-set policy overrides are enabled. If the controller ignores
that setting, stop and have the installation owner configure a compatible policy.
For Azure, make required installation changes through Terraform/the extension's
supported configuration, never a GitOps-managed Argo CD ConfigMap or replacement
Helm release. No controller installation changes are made by this repository refactor.

Set `CONTEXT` to the intended kubeconfig context and choose exactly one pair:

```sh
# Azure dev market (extension-managed)
CLUSTER=clusters/azure/DEV-JKS/dev/eus2/spoke-atlas/aks-atlas-market
ROOT=azure-dev-uksouth-atlas-market

# Local kind (Helm-based)
CLUSTER=clusters/local/kind/kind-platform
ROOT=local-kind
```

The Azure directories and cluster overlays identify `eus2`. Earlier root and child
source paths incorrectly used nonexistent `uksouth` directories. Paths now use
`eus2`; existing Application names retain `uksouth` to preserve their identity.
This does not change the location of any cloud resource. AWS has reserved files
only and no root to apply.

For local kind, create the cluster separately with `kind create cluster --name
kind-platform`; its context is `kind-kind-platform`. For Helm-based installations
only, install Argo CD with the same chart and values selected by `platform-argocd`:

```sh
helm repo add argo https://argoproj.github.io/argo-helm
helm repo update argo
helm upgrade --install argocd argo/argo-cd --version 7.8.13 \
  --kube-context "$CONTEXT" --namespace argocd --create-namespace \
  -f catalog/platform/argocd/values.yaml --wait
```

Append any cluster Argo CD values override to both the Helm command and the child
Application. Do not run this Helm command on extension-managed clusters.

There is no bootstrap script. From the repository root, apply only the selected
cluster's initial root after verifying the context:

```sh
kubectl --context "$CONTEXT" cluster-info
kubectl --context "$CONTEXT" apply -f "$CLUSTER/bootstrap/root.yaml"
kubectl --context "$CONTEXT" -n argocd port-forward service/argocd-server 8080:443 &
PORT_FORWARD_PID=$!
```

`argocd/kustomization.yaml` selects projects and children, never `bootstrap/root.yaml`.
Roots do not manage themselves. Keep their revisions aligned with child revisions.

Keep this shell open. Log in with `argocd login localhost:8080` and the
bootstrap administrator credentials. The initial certificate is self-signed; inspect it before
accepting the CLI's certificate prompt. Retrieve the initial password through your normal secure
operator workflow. Rotate the bootstrap password and configure access controls before sharing access.

## Reconcile in order

If the cluster was bootstrapped before the directory rename, reapply its matching root manifest after
the new paths reach the tracked Git revision. This updates the root's source path. Sync the root to
update child Application paths before syncing children. Root and child Application names remain the
same; do not delete and recreate them for a directory move.

If an Azure cluster already has a `platform-argocd` Application from the old bundle,
removing it from Git does not delete it: root pruning remains disabled. Review its
live ownership and finalizers before retiring that obsolete Application without
cascading deletion. Do not prune or delete extension-owned Argo CD resources.

1. Sync the root with `argocd app sync "$ROOT"`. Verify AppProjects and child Applications appear.
   A root sync does not sync its children; sync waves are not a dependency scheduler here.
2. Sync `platform-namespaces`, then wait for its sync to complete.
3. For Helm-based installations only, sync `platform-argocd`. Skip this step on extension-managed Azure clusters. This hands resource reconciliation to Argo CD using the same chart,
   release name, namespace, and values as bootstrap. Do not subsequently run Helm upgrade or
   uninstall against this release: stale Helm release metadata is not the source of truth.
4. Sync `platform-cert-manager`, `platform-external-secrets`, `platform-monitoring`, and
   `platform-gateway-controller`. Wait for controllers, webhooks, and CRDs to become ready.
5. Sync `platform-policy`, then `platform-gateway-resources`. Confirm the GatewayClass is accepted
   and the Gateway is programmed. A kind Gateway can remain pending until load-balancer support exists.
6. Follow [onboarding](onboarding-an-application.md) to configure workload images and replicas before
   syncing `atlas-ml-inference` or the dedicated cluster's `atlas-market-api` and `atlas-market-worker`.
   Local kind has no workload Applications by default.

For example, sync and wait for a controller with:

```sh
argocd app sync platform-cert-manager
argocd app wait platform-cert-manager --sync --health --timeout 300
```

The intended result is synced Applications with healthy controllers. Zero-replica workloads do not
serve traffic. HTTPRoutes use example DNS names; configure DNS and TLS separately before real use.
If a sync reports a missing custom resource kind, confirm the owning controller's CRDs are established,
then refresh and retry the dependent Application. See [recovery](recovery.md) for other failures.

After finishing, stop the background port-forward with `kill "$PORT_FORWARD_PID"`.

## Migrate existing platform Applications

Do this handover once per already-bootstrapped cluster. Offline rendering proves
manifest compatibility, not controller adoption. Keep the previous Git revision
available and perform the checks below in the intended context.

1. Before root sync, save the existing platform Applications, UIDs, owner references,
   finalizers, and Argo CD tracking annotations/labels. Confirm no other
   ApplicationSet already owns them. This repository previously supplied no
   deletion finalizers; investigate any live differences before proceeding.
2. Confirm the ApplicationSet controller honors `create-update` as described above.
   Merge catalog component inputs and cluster selections together to the tracked
   `main` revision. The Git-file generator reads the remote revision, not a local
   checkout. Public repository access requires no credential Secret.
3. Sync the root **without pruning**. The root now manages the `platform`
   ApplicationSet instead of the platform Application manifests. Existing
   Applications must remain present while the controller adopts matching names.
   Wait for ApplicationSet conditions to report successful parameter generation
   and updates. Check every Application retains its original UID, spec, namespace,
   and annotations, and gains an owner reference to this cluster's `platform`
   ApplicationSet. A recreated UID is a failed handover, not a successful adoption.
4. Verify no generated Application has acquired a resource-deletion finalizer.
   `preserveResourcesOnDeletion: true` requests that behavior; it does not remove
   an unexpected pre-existing finalizer. Do not delete or prune resources to fix
   a failed adoption.
5. After verifying ownership, remove only obsolete **root** resource-tracking
   annotations/labels from these generated Applications if they remain. Inspect
   the installation's tracking method and values first: commonly
   `argocd.argoproj.io/tracking-id`, `argocd.argoproj.io/installation-id`, or the
   configured instance-label key. Do not remove unrelated labels or the preserved
   sync-wave annotation. Confirm the root no longer reports those Applications
   as resources eligible for pruning. Ordinary workload Applications stay root-owned.
6. Refresh Applications and review diffs before any manual child sync. No workload
   manifest or effective Helm value change is expected from this handover.

If adoption fails, pause root/child syncs and inspect controller conditions and
ownership conflicts. Restore the prior root resources without pruning. Before
removing an ApplicationSet, coordinate controller reconciliation and inspect its
owner references: Kubernetes garbage collection can delete owned Applications
even when `applicationsSync` prevents generator-driven deletion. Use orphan
propagation only in an approved recovery procedure, after verifying finalizers;
never cascade deletion during this migration.

Removing a component from the list intentionally does not automatically retire
its Application when `create-update` is honored. Review and explicitly retire the
Application separately. `preserveResourcesOnDeletion` protects workloads from
ApplicationSet deletion; it is not a substitute for checking live finalizers.
