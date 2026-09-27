# Adding a cluster or component

Read [configuration boundaries](configuration-boundaries.md) for defaults and
permission ownership. Run commands from the repository root. The examples below
are configuration changes, not deployment commands.

## Add one component to an existing cluster

Select an existing entry from `catalog/platform/components/` in the cluster's
ApplicationSet list. Include its required namespaces and AppProject permissions.
Only add a new catalog entry when a cluster needs it; do not add future placeholders.
Run `scripts/validate.sh` and update the cluster selection assertions when needed.

UK South uses the dedicated `catalog/platform/cert-manager` Application instead of
the broader ApplicationSet. Do not select both arrangements on the same cluster.
Its chart and issuer must stay in one sync operation; see [HTTPS](https.md).

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
