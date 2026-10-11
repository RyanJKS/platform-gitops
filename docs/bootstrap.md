# Bootstrap Argo CD and transfer ownership

Terraform installs a basic Argo CD chart and creates the `all-apps` root
Application in the built-in `default` project. The cluster root creates three
AppProjects and five child Applications. Argo CD's self-management child uses
the GitOps-owned `bootstrap` project and stays manually synced until ownership
handoff. Each child renders its own chart and optional configuration.

## Terraform bootstrap inputs

The existing sibling infrastructure repository is `platform-infrastructure`.
Its AKS AGIC units are under
`infrastructure/azure/live/DEV-JKS/dev/uks/sandbox/platform/aks-agic/`.

| Unit | Input | Purpose |
| --- | --- | --- |
| `argocd_helm_manifest` | Hardcoded `release` in Terragrunt, chart defaults | Installs `argocd` in namespace `argocd`, chart `argo-cd` version `10.9.0` |
| `argocd_application_manifest` | Inline manifest in Terragrunt | Creates `all-apps`, targeting the cluster root on `dev/sandbox` |

The Helm unit does not read GitOps chart metadata or runtime values. The module's
default empty values list lets the official chart supply installation defaults,
including CRDs. Namespace creation is enabled, installation waits for readiness,
and its timeout is 600 seconds. There is no bootstrap ingress or HA override.

Repository URLs live in infrastructure
`infrastructure/azure/_envcommon/repo-urls.hcl`. The root uses its configured
GitOps URL and selects `clusters/azure/DEV-JKS/dev/uks/sandbox/aks-agic` on
`dev/sandbox`. Neither active bootstrap unit reads a local GitOps manifest.

The local Argo CD overlay imports the catalog Kustomization, which includes
`root_project.yaml` and the self-management Application. The root creates
`bootstrap`, `platform-apps`, and `apps` at wave `-4`, before child Applications.
Treat root repository write access as administrative access because the root
can create projects and privileged Applications.

## Repository access

`RyanJKS/platform-gitops` is public. Set infrastructure
`platform_gitops_github_url` to `https://github.com/RyanJKS/platform-gitops.git`,
matching the catalog child sources. Public HTTPS needs no repository credential.
The Terraform module repository can continue using SSH on the runner; those
credentials are not available inside Argo CD's repo-server.

An SSH root URL without an Argo CD repository key fails with
`SSH agent requested but SSH_AUTH_SOCK not-specified`. Changing the infrastructure
URL only updates configuration files: plan and apply `argocd_application_manifest`
to update an existing root Application, then refresh `all-apps`.

## Prepare the AKS AGIC sandbox

Create AKS, networking, Application Gateway, DNS zones, identities, federated
credentials, and role assignments through infrastructure first. Authenticate the
deployment identity with Azure CLI and grant it Kubernetes access. Install
Terraform, Terragrunt v1.1.5, Azure CLI, and `kubelogin`; the runner must reach AKS.

Generated Helm v3 and Kubernetes providers use `kubelogin get-token --login
azurecli` directly. `az aks get-credentials` writes kubeconfig and is not an exec
credential command. AKS local accounts are disabled.

Before creating the root, validate the cluster Kustomizations, fill deployment
placeholders, configure Argo CD repository access if needed, and publish the
reviewed configuration to `dev/sandbox`.

```sh
CLUSTER=clusters/azure/DEV-JKS/dev/uks/sandbox/aks-agic
rg -n 'REPLACE_PLACEHOLDER' "$CLUSTER"
kubectl kustomize "$CLUSTER" > /tmp/aks-agic-applications.yaml
kubectl kustomize "$CLUSTER/platform/cert-manager/configs"
kubectl kustomize "$CLUSTER/platform/external-dns/configs"
kubectl kustomize "$CLUSTER/workloads/chaos-generator/configs"
```

The root should render AppProjects and child Applications. Each child renders
its own workloads/controller resources. Nested Kustomizations flatten one render;
directory nesting does not create Application sync boundaries.

## Initial installation stages

Run from the infrastructure repository with AKS and authentication ready:

```sh
AKS_AGIC_DIR=infrastructure/azure/live/DEV-JKS/dev/uks/sandbox/platform/aks-agic
(cd "$AKS_AGIC_DIR/argocd_helm_manifest" && ARGOCD_BOOTSTRAP=true terragrunt run -- plan)
(cd "$AKS_AGIC_DIR/argocd_helm_manifest" && ARGOCD_BOOTSTRAP=true terragrunt run -- apply)
(cd "$AKS_AGIC_DIR/argocd_application_manifest" && terragrunt run -- plan)
(cd "$AKS_AGIC_DIR/argocd_application_manifest" && terragrunt run -- apply)
```

Install Helm before planning the root manifest unit: the Kubernetes provider
needs Argo CD CRDs during planning. Keep initial stages separate and verify
readiness before continuing. The root unit declares a Helm dependency for Terragrunt ordering and waits
for the Application CRD using its generated kubeconfig before plan/apply.

The Helm unit is excluded from routine `run --all` operations, and direct
plan/apply/destroy/import/refresh commands are guarded. Use `ARGOCD_BOOTSTRAP=true`
only for initial installation or reviewed recovery, never after handoff.
Terraform continues owning only the root Application after Helm handoff.

## Seed the root and access Argo CD

The root is seeded by the second Terraform stage; there is no extra root YAML to
apply with kubectl. During the default bootstrap, Argo CD serves HTTPS internally.
With the intended AKS context selected:

```sh
kubectl -n argocd get applications,appprojects
kubectl -n argocd rollout status deployment/argocd-server --timeout=300s
kubectl -n argocd rollout status deployment/argocd-repo-server --timeout=300s
kubectl -n argocd port-forward service/argocd-server 8080:443
```

Open `https://localhost:8080` using the installation's initial TLS certificate.
Retrieve the initial admin password locally or use configured SSO; do not commit
credentials. After the Azure runtime profile is applied, Application Gateway
terminates public TLS and forwards HTTP to Argo CD. The intended runtime hostname
is `argocd.jkslabs.site` and certificate Secret is `argocd-ingress-tls`.
Follow [public access debugging](debugging.md#check-public-argo-cd-access).

## Transfer Helm resource ownership to Argo CD

The child must remain manually synced until Terraform relinquishes its release.
The minimal installation and runtime configuration intentionally differ: the
first GitOps sync changes sizing, Redis, ingress, and other runtime settings.
Keep release name and namespace `argocd` and review chart/image versions and the
full diff before adoption. Existing extension/raw-manifest installations need a
separate migration; this handoff assumes the official chart is already installed.

1. Pause jobs that can modify this release. Confirm the root has created child
   `argocd`, and review its diff against the installed chart resources.
2. Back up the Helm unit's Terraform state and verify that it contains the expected
   release. From that unit directory, these operator commands remove state
   ownership without uninstalling resources:

   ```sh
   umask 077
   STATE_BACKUP=$(mktemp /tmp/argocd-helm-state.XXXXXX)
   terragrunt run -- state pull > "$STATE_BACKUP"
   terragrunt run -- state list
   terragrunt run -- state rm helm_release.this
   terragrunt run -- state list
   ```

   Securely retain the validated backup. **Do not run Terraform destroy or Helm
   uninstall to transfer ownership.**
3. Manually sync the Argo CD child and verify `Synced`, `Healthy`, controller/server
   readiness, and continued API/UI access. Investigate immutable selectors,
   generated Secrets, or unexpected deletions before allowing the rollout.
4. Remove the cluster patch disabling the child's automatic sync and publish it.
   Catalog defaults enable automatic sync and self-heal; pruning stays disabled.
   Leave `ARGOCD_BOOTSTRAP` unset in subsequent infrastructure runs.

Afterward, the child merges runtime values in this order:

1. `catalog/platform/argocd/values.yaml`.
2. `catalog/platform/argocd/azure-values.yaml`.
3. Cluster `helm.valuesObject` from `platform/argocd/application-patches.yaml`.

Later maps override earlier maps; later lists replace earlier lists. Argo CD
templates Helm manifests and reconciles Kubernetes resources; it does not update
the original Helm release record, which may remain stale. Future chart/value
changes go through GitOps. `ignore_changes` does not relinquish Helm ownership.

## Sync the children and verify deployment

Verify children in wave order: AGIC, External DNS, cert-manager, then
chaos-generator. Check AGIC controller readiness and Application Gateway access
before checking External DNS workload identity and zone discovery. Check issuer
readiness and workload certificates next. Apply Argo CD runtime configuration only
after the ownership handoff above.

The full runtime values include Application health customization for root waves.
The minimal bootstrap does not include that customization, so initial waves order
child definitions without proving controller readiness; verify children manually.
Existing children reconcile independently of later root syncs.

Runtime values select two server/repo-server replicas, one controller with `250m`
CPU and `1Gi` memory requests and `2Gi` memory limit, and Redis HA. Redis HA requires
at least three schedulable nodes. Verify capacity and TLS/DNS before public access.
See [debugging](debugging.md) for controller and certificate checks.

## Recovery

Keep a known-good Git commit and AKS access independent of Argo CD. If it can
reconcile, revert the bad runtime change in Git. Otherwise render and apply the
reviewed known-good chart/manifests with kubectl to restore the controller, then
reconcile that revision. Do not use the stale Helm record for an unreviewed
rollback or uninstall. Restoring Terraform ownership requires a separate reviewed
migration with the child suspended. No live installation, state removal, or
recovery is performed by this repository change.
