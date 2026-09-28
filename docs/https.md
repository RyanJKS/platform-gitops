# UK South HTTPS

## Scope and bootstrap ordering

The root entry point is
`clusters/azure/DEV-JKS/dev/uks/atlas/aks-shared/argocd/kustomization.yaml`.
It imports `catalog/platform/cert-manager` and preserves the chaos-generator
Application and its existing overlay. No additional ingress controller, Argo CD
installation, ExternalDNS, Key Vault integration, or CSI driver is selected.

The dedicated `cert-manager` AppProject permits the official Jetstack chart and
this Git repository. Destinations are `cert-manager` and `kube-system` (the chart's
leader-election Role and RoleBinding). Cluster permissions cover Namespaces, CRDs,
ClusterRoles, ClusterRoleBindings, admission webhooks, and ClusterIssuers. The
Terraform-owned root uses `default`; that project's existing permissions must allow
Applications and AppProjects in `argocd`. Azure extension cluster RBAC must also
permit the chart's resources; AppProjects cannot grant Kubernetes RBAC.

One Application renders the chart and issuer from two sources. Argo CD 2.6 or later
is required for multiple sources and external Helm value files. Git sources use
`main`; change all relevant root and child revisions together for branch testing.
The root's project wave only creates the AppProject before the Application. It is
not the cert-manager readiness mechanism.

Within the child sync:

1. Wave 0 applies chart CRDs, controller, cainjector, webhook, and startup-check
   ServiceAccount/RBAC. Argo CD waits for Deployment health before proceeding.
2. Wave 1 runs the chart's own startup API check as an Argo CD **Sync** hook. The
   check waits up to five minutes per attempt for cert-manager API admission,
   including webhook readiness. Failure blocks later waves and fails the sync.
3. Wave 2 applies `letsencrypt-prod`. Argo CD skips initial missing-CRD dry-run
   automatically because the matching CRD is included in the same sync.

The default Helm post-install startup hook would map to PostSync and run after
issuer creation. `argocd-values.yaml` removes those Helm hooks, keeps supporting
RBAC as ordinary resources, and moves the check into Sync. Successful Jobs are
cleaned up; failed Jobs remain for diagnostics until the next sync replaces them.
Automated sync retries up to five times with backoff. After exhaustion, repair the
underlying problem and retry the Application sync. Selective resource sync skips
hooks; use a full Application sync when installing or upgrading this component.

## Version compatibility

The official chart is pinned to `v1.20.4`, with `crds.enabled: true`, release name
`cert-manager`, and destination namespace `cert-manager`. Upstream currently
[supports cert-manager 1.20 on Kubernetes 1.32–1.35](https://cert-manager.io/docs/releases/).
This was checked on 2026-09-27. Chart metadata's broader minimum Kubernetes version
is not the supported-version matrix. The existing market chart pin is
unchanged; the new Application is selected only by UK South.

The adjacent infrastructure checkout's
`aks_cluster/terragrunt.hcl` under the UK South Atlas platform configuration
leaves `kubernetes_version` commented out. Its example `1.37` is not an active
setting. Azure therefore selects the regional default; no actual cluster version
was available from repository configuration. Before rollout, inspect
`kubectl version -o yaml` or `az aks show --resource-group "$RESOURCE_GROUP" --name
"$CLUSTER_NAME" --query currentKubernetesVersion -o tsv`. If outside 1.32–1.35,
select a supported cert-manager release and rerender before enabling reconciliation.
Do not downgrade an existing cluster to match the offline validation baseline.

## Extension settings outside this repository

Read-only inspection found the extension configuration in the adjacent
`platform-infrastructure` checkout, in `aks_argocd_extension/terragrunt.hcl` under
the UK South Atlas platform configuration. Its directory layout is maintained
separately from this repository.

It selects `Microsoft.ArgoCD`, name `argocd-ext`, release train `preview`, with no
explicit extension version. It already configures the hostname, ingress class,
TLS, and HTTPS backend protocol. It does not configure the ClusterIssuer annotation.
Update its existing `inputs.extension.configuration_settings` map as follows,
preserving unrelated settings:

```hcl
"global.domain"                     = "argocd.jkslabs.site"
"configs.cm.url"                    = "https://argocd.jkslabs.site"
"server.service.type"               = "ClusterIP"
"server.ingress.enabled"            = "true"
"server.ingress.hostname"           = "argocd.jkslabs.site"
"server.ingress.ingressClassName"    = "webapprouting.kubernetes.azure.com"
"server.ingress.tls"                = "true"
"server.ingress.annotations.cert-manager\\.io/cluster-issuer" = "letsencrypt-prod"
"server.ingress.annotations.nginx\\.ingress\\.kubernetes\\.io/backend-protocol" = "HTTPS"
```

The doubled backslashes are HCL escapes; the resulting single backslashes protect
dots in Helm annotation keys. Keep the server's HTTPS backend enabled. No
`server.insecure` change is required for this configuration.

[Microsoft documents](https://learn.microsoft.com/en-us/azure/azure-arc/kubernetes/tutorial-use-gitops-argocd)
a breaking configuration change at extension `1.0.0-preview`, which adopts the
upstream Argo CD chart. These keys require that generation or a later compatible
extension, and bundled Argo CD must support multiple sources. Old `0.0.x` extensions
need Microsoft's documented migration; do not apply these keys blindly or assume
the unpinned preview train proves the installed version. Inspect it first:

```sh
az k8s-extension show --cluster-type managedClusters \
  --cluster-name "$CLUSTER_NAME" --resource-group "$RESOURCE_GROUP" \
  --name argocd-ext --query '{version:version,settings:configurationSettings}'
```

Confirm the installed extension chart renders `spec.tls` with host
`argocd.jkslabs.site` and Secret `argocd-server-tls`, the upstream chart's fixed name
for `server.ingress.tls=true`. Pin a verified compatible extension version through
`inputs.extension.version` if required by infrastructure policy; its blueprint
passes that value to `azurerm_kubernetes_cluster_extension.version`. No extension
version was invented or applied here. Do not add a second Ingress, Certificate, or
Secret to Git; ingress-shim creates the Certificate from the annotated Ingress.

The sibling `argocd_project_manifest/terragrunt.hcl` creates the root Application
`all-apps` (despite the unit name). Change its
`inputs.manifest.spec.source.targetRevision` from `feature/repo-setup` to `main`
after merge. Its interpolated UK South source path already matches this repository.
The chaos child already uses `main`, as does the new cert-manager Git source.
These external Terraform/Terragrunt changes were **not applied** by this task.

## DNS, issuance, and readiness

GoDaddy must delegate `jkslabs.site` to the Azure DNS zone's nameservers. AKS
application routing must have that zone attached and its managed identity must
retain DNS Zone Contributor. Its existing DNS integration manages ingress records;
this repository does not deploy another ExternalDNS controller. Verify public A
records resolve to the routing load balancer and that any AAAA record also reaches
the solver. NS delegation alone does not create an application record.

Allow inbound TCP 80 to the solver and TCP 443 for HTTPS. Keep port 80 reachable
for renewal. Permit controller DNS resolution and outbound HTTPS to Let's Encrypt.
If CAA records exist, allow `letsencrypt.org`. HTTP-01 uses
`webapprouting.kubernetes.azure.com`; no Azure credentials are needed by cert-manager
for this solver. Validate the actual DNS and routing configuration before expecting
issuance; neither was queried live in this task.

`letsencrypt-prod` uses the production ACME endpoint. No real ACME email was found
in the inspected configuration, so the optional field is omitted. Add a real
address at `spec.acme.email` in
`catalog/platform/cert-manager/issuers/letsencrypt-prod.yaml` if desired.
Cert-manager creates the account Secret `letsencrypt-prod-account-key` in its
cluster resource namespace (`cert-manager`) and certificate Secret
`argocd-server-tls` in `argocd`. Never commit their contents.

Run the readiness commands in the README after rollout. `ClusterIssuer` must report
`Ready=True`; the `argocd-server-tls` Certificate must then report `Ready=True`.
Inspect `kubectl -n cert-manager logs job/cert-manager-startupapicheck` if bootstrap
stops at wave 1. A missing Job after success is expected. Inspect Certificate
conditions, CertificateRequests, Orders, Challenges, and temporary solver Ingresses
for issuance failures. Fix DNS or port-80 access before repeatedly retrying the
production endpoint; Let's Encrypt rate limits apply. Finally verify the served
certificate with `curl -I https://argocd.jkslabs.site`.

## Pruning and cleanup

The cert-manager Application intentionally enables self-healing without automatic
pruning and has no cascading-deletion finalizer. Removing its selection therefore
does not automatically uninstall cert-manager or delete issuer/account resources.
Chart resource retirement requires an explicit reviewed operation. Do not add a
second `platform-cert-manager` Application or select the older ApplicationSet
component on UK South; first reconcile any existing release ownership.

Terraform's root has automatic pruning and a cascading finalizer. Review its live
tracked resources before changing paths or enabling the new selection. Removing
an AppProject while Applications still use it also requires review. These files do
not prove what a cluster currently runs; no live ownership was inspected.

The legacy platform catalog bundles and GitHub runner examples have been removed.
Their obsolete Azure selections and Kustomizations are also removed. Market retains
its workload Applications, AppProject, and namespace. See
[legacy migration](bootstrap.md#migrate-existing-platform-applications) before
syncing an existing market root, and [local kind migration](local-kind.md#migrate-an-existing-kind-installation)
for the separate local simplification. UK South and chaos-generator are unchanged.
