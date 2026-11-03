# AKS Envoy platform applications

The Helmfile root selects this directory after bootstrap. It creates the
selected Envoy Gateway namespace at wave `-6` and the `platform-apps` AppProject at
wave `-5`, then installs shared Gateway API CRDs and cert-manager at wave `-4`,
Envoy Gateway at `-3`, and ExternalDNS at `-2`. GatewayClass, Gateway, ClusterIssuer, Certificate,
and Argo CD HTTPRoutes reconcile after the controllers. Bootstrap releases remain
owned by Helmfile. The Gateway API Application imports its catalog definition;
Argo CD applies the seven selected upstream CRDs directly with server-side apply.
The existing Application health check gates Envoy Gateway on child readiness.
Gateway resources skip dry-run only while their CRDs are missing on first sync.

`namespaces/kustomization.yaml` selects reusable namespace bases from
`catalog/platform/namespaces/` for Envoy Gateway. Add only namespaces needed by
this root. Bootstrap retains ownership of `argocd`, `cert-manager`,
`external-dns`, `external-secrets`, and resources in `kube-system`; do not add those
namespaces to this selection. Child Applications retain `CreateNamespace=true`
so they can also be selected without this namespace layer.

ExternalDNS imports its Azure ConfigMap from the catalog and applies
`external-dns/configmap-patches.yaml` for this cluster's Azure settings. The
ConfigMap uses wave `-5`, after namespaces and before the controller Application.

Both DNS controllers consume named Secrets created by the central
[platform secrets configuration](../../bootstrap/external-secrets-config/values.yaml).
The bootstrap release owns the ExternalSecrets and their target namespaces, so
they exist before the root selects controllers. cert-manager reads
`certmanager-client-id` in `cert-manager`; ExternalDNS reads
`externaldns-client-id` in `external-dns`. Both use key `client-id` through
`AZURE_CLIENT_ID`. Tenant and DNS resource settings remain in cluster overlays.

## Terraform inputs

Provision two workload UAMIs with the AKS OIDC issuer and audience
`api://AzureADTokenExchange`:

- ExternalDNS: subject `system:serviceaccount:external-dns:external-dns`.
  Store its client ID in Key Vault as `externaldns-client-id`.
- cert-manager: subject `system:serviceaccount:cert-manager:cert-manager`.
  Store its client ID in Key Vault as `certmanager-client-id`.

Both UAMIs need DNS Zone Contributor on the public `jkslabs.site` DNS zone.
ExternalDNS also needs Reader on the DNS resource group for discovery. Assign
roles to UAMI principal IDs; the ESO-managed Secrets contain client IDs. The
`platform-secrets` reader needs secret-read access to both keys, and the store
must allow both namespaces. After a client ID changes, restart its controller
Deployment once the ExternalSecret has refreshed; container environment variables
do not change automatically.
The control-plane UAMI handles AKS infrastructure and network resources. Envoy
Gateway needs no workload UAMI for this configuration.

Review the DNS subscription, resource group, tenant, domain, and existing record
ownership. ExternalDNS uses owner ID `dev-uks-sandbox-aks-envoy`; it does not take
over records owned by another cluster. cert-manager uses production ACME DNS-01
and writes challenge TXT records with its own identity.

## Argo CD access

The Gateway terminates HTTPS for `argocd.jkslabs.site` using Certificate
`argocd-server`. HTTP redirects to HTTPS. The HTTPS route forwards to the internal
`argocd-server:80` Service; bootstrap sets `server.insecure: true`. The
ServiceAccount and SSO configuration remain in bootstrap. Use `--grpc-web` for
the Argo CD CLI through this route.

Both listeners accept routes only from `argocd`. Add namespace permissions and
listener hostnames deliberately when extending the Gateway for other workloads.
The Envoy Application's values revision matches the root's `feature/monitoring`;
update both together when moving branches.

Render from the repository root:

```sh
kubectl kustomize clusters/azure/DEV-JKS/dev/uks/sandbox/aks-envoy/apps
```

See the [full deployment guide](../../../../../../../../../docs/helm-bootstrap.md#platform-applications-and-azure-identities)
for the Terraform handoff, sync order, DNS-01 configuration, and verification.
