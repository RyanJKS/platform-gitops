# Bootstrap

## Azure shared bootstrap

For the `uks` and `eus2` variants of `spoke-atlas/aks-shared`, Terraform installs
Argo CD through the Azure extension and creates the root Application `all-apps`
in namespace `argocd`. Do not apply the inactive legacy `bootstrap/root.yaml`,
install another Argo CD instance, or create another root owner.

Configure Terraform's root with this exact source and enable automated sync:

```yaml
spec:
  source:
    repoURL: https://github.com/RyanJKS/platform-gitops.git
    targetRevision: main
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

Commit and push these changes to `main` before Argo CD can read them.
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
NGINX Ingress and the commented probes targeting port 3000. UK South also selects
cert-manager and its dedicated AppProject; see [HTTPS bootstrap](https.md).
East US 2 still selects only chaos-generator. No new ingress controller is installed.
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
to `main`; the child's automatic sync applies it. Manual scaling can
be reverted by self-healing. Terraform's root still watches the `uks/…/argocd`
entry point; only the child source points at the workload overlay.

### Review pruning before applying

The `uks` entry point adds cert-manager to its existing chaos-generator selection.
The cert-manager child does not enable pruning or cascading deletion; see
[HTTPS pruning](https.md#pruning-and-cleanup). If an existing root is repointed
to this path, review the resources that root currently tracks before pruning. The previous `eus2` entry point rendered AppProjects `atlas-ml` and `platform`, Application
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

## Market example

Local kind has its own [bootstrap and migration guide](local-kind.md).
The remaining Azure market root selects only its namespace, workload AppProject,
and API/worker Applications. Terraform must already provide Argo CD. From the
repository root, after changes reach `main`:

```sh
CLUSTER=clusters/azure/DEV-JKS/dev/eus2/spoke-atlas/aks-atlas-market
kubectl --context "$CONTEXT" apply -f "$CLUSTER/bootstrap/root.yaml"
```

The root retains the legacy name `azure-dev-uksouth-atlas-market` despite its `eus2`
source path. Verify the intended context; naming does not move cloud resources.
Sync the root without pruning, then sync the workload Applications after configuring
real images and replicas. They retain manual sync and zero-replica example defaults.
The API HTTPRoute requires externally supplied Gateway API CRDs and a compatible
Gateway/controller. The removed platform bundles no longer provide these resources.

## Migrate existing platform Applications

The platform ApplicationSet and its catalog bundles have been removed. Before
syncing a previously deployed market root, inspect its tracked resources, the old
`platform` ApplicationSet, child Applications, owner references, and finalizers.
Keep pruning disabled during this review. The old ApplicationSet may still control
children even though it is no longer selected in Git.

Retire the old ApplicationSet with orphan propagation only after reviewing its
live ownership; preserve generated Applications and workloads while deciding their
fate individually. A resource-deletion finalizer can cause deletion of a child's
workloads. Namespace deletion can remove unrelated resources. Do not cascade-delete
an ApplicationSet or namespace bundle as a shortcut for this migration.

Market now selects `atlas-market` directly from the shared namespace catalog.
Before root adoption, detach this namespace from the former `platform-namespaces`
Application's tracking without deleting it, and retire that former owner so it
cannot reclaim the namespace. Preserve existing namespace policy labels until
reviewed separately. Workload AppProject and Application identities remain unchanged.

Review remaining legacy platform Applications individually. Keep needed live
services until their replacement or retirement is explicitly planned. Remove the
old platform AppProject only when no Applications still use it. This repository
change does not perform any live uninstall, namespace migration, or ownership transfer.
