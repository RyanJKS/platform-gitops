# Adding a cluster or component

Read [configuration boundaries](configuration-boundaries.md) for defaults and
permission ownership. Run commands from the repository root. The examples below
are configuration changes, not deployment commands.

## Add one component to an existing cluster

For example, add the optional GitHub Actions controller to local kind. Its chart
version matches the existing optional ARC example; this does not create runners
or require storing credentials in Git.

1. Add `catalog/platform/components/github-actions-controller.yaml`:

   ```yaml
   name: github-actions-controller
   type: helm
   wave: '-2'
   namespace: arc-systems
   repoURL: ghcr.io/actions/actions-runner-controller-charts
   chart: gha-runner-scale-set-controller
   version: 0.13.0
   releaseName: arc-controller
   defaultValues: $values/catalog/platform/github-actions/controller/values.yaml
   ```

2. In `clusters/local/kind/kind-platform/argocd/kustomization.yaml`, append
   `- component: github-actions-controller` to the list patch's `value` array.
   No Application manifest or template copy is needed.
3. Add `../../../../../../catalog/platform/github-actions/controller/namespaces`
   to the resources in that cluster's `platform/namespaces/kustomization.yaml`.
4. In that cluster's `argocd/projects/kustomization.yaml`, append these operations
   to the existing platform project's patch:

   ```yaml
   - op: add
     path: /spec/sourceRepos/-
     value: ghcr.io/actions/actions-runner-controller-charts
   - op: add
     path: /spec/destinations/-
     value:
       server: https://kubernetes.default.svc
       namespace: arc-systems
   ```

5. Run `scripts/validate.sh`. Update the explicit component-count assertion in
   `scripts/render-applications/main_test.go` to reflect the reviewed addition.
   Merge, sync the root, wait for the generated Application, sync namespaces, and
   then sync the controller when ready. Follow [runner onboarding](github-actions-runners.md)
   separately before adding a runner scale set. Do not also select the ordinary
   example controller Application: one owner per Application.

For an already-cataloged component, start at step 2. To override values, add a
cluster file and an `overrideValues` list to its element; paths are relative to the
one configured `clusterPath`. To stage a chart upgrade, set `versionOverride` on
that element. Keep unique workload Applications ordinary when a template adds no
useful reuse.

## Add a cluster without copying shared manifests

The existing AWS reserved directory is a concrete example. After provisioning its
cluster and Argo CD separately, the following four files select only namespaces
and cert-manager. The example assumes Argo CD is already installed and deliberately
does not select the `argocd` component. Choose installation ownership explicitly.

Create `clusters/aws/personal-account/dev/eu-west-2/spoke-atlas/eks-shared/bootstrap/root.yaml`:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: aws-dev-eu-west-2-shared
  namespace: argocd
spec:
  project: default
  source:
    repoURL: https://github.com/RyanJKS/platform-gitops.git
    targetRevision: main
    path: clusters/aws/personal-account/dev/eu-west-2/spoke-atlas/eks-shared/argocd
  destination:
    server: https://kubernetes.default.svc
    namespace: argocd
  syncPolicy:
    syncOptions:
      - ServerSideApply=true
```

Create that cluster's `argocd/kustomization.yaml`:

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  - ../../../../../../../../catalog/platform/applicationsets/platform
  - projects
patches:
  - target:
      kind: ApplicationSet
      name: platform
    patch: |
      - op: replace
        path: /spec/generators/0/matrix/generators/0/list/elements
        value:
          - component: namespaces
          - component: cert-manager
      - op: replace
        path: /spec/generators/0/matrix/generators/1/git/values/clusterPath
        value: clusters/aws/personal-account/dev/eu-west-2/spoke-atlas/eks-shared
```

Create `argocd/projects/kustomization.yaml` in the same cluster:

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  - ../../../../../../../../../catalog/platform/projects/platform
```

Create its `platform/namespaces/kustomization.yaml`:

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  - ../../../../../../../../../catalog/platform/namespaces/base
```

This imports namespace identities only; it does not select quota, Gateway, workload
policy, or business applications. There are no cluster value files because this
selection has no differences from defaults. Remove obsolete reserved README text
when implementing the cluster and add its selection to the validation tests.

Run `scripts/validate.sh`, review the generated Applications, merge configuration
to the tracked revision, and follow [bootstrap](bootstrap.md). After verifying the
intended context, the initial command would be:

```sh
kubectl --context "$CONTEXT" apply -f clusters/aws/personal-account/dev/eu-west-2/spoke-atlas/eks-shared/bootstrap/root.yaml
```

The root remains outside its own Kustomization. Each Argo CD instance consumes
only its explicit list and cluster path. Preserve the shorter local kind hierarchy;
do not add Azure-specific path segments. Do not create empty placeholder folders
or use an empty Kustomization to retire a deployed cluster.
