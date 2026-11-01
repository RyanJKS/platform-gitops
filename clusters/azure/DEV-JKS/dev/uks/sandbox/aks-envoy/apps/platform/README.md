# AKS Envoy platform applications

The Helmfile root selects this directory after bootstrap. It creates the
`platform-apps` AppProject, then installs cert-manager, Envoy Gateway, and
ExternalDNS in sync waves. GatewayClass, Gateway, ClusterIssuer, Certificate,
and Argo CD HTTPRoutes reconcile after the controllers. Shared Gateway API
CRDs and bootstrap releases remain owned by Helmfile.

## Terraform inputs

Provision two workload UAMIs with the AKS OIDC issuer and audience
`api://AzureADTokenExchange`:

- ExternalDNS: subject `system:serviceaccount:external-dns:external-dns`.
  Set `REPLACE_EXTERNAL_DNS_UAMI_CLIENT_ID` in
  [external-dns/application-patches.yaml](external-dns/application-patches.yaml).
- cert-manager: subject `system:serviceaccount:cert-manager:cert-manager`.
  Set the same `REPLACE_CERT_MANAGER_UAMI_CLIENT_ID` in
  [cert-manager/kustomization.yaml](cert-manager/kustomization.yaml) and
  [cert-manager/issuer-patches.yaml](cert-manager/issuer-patches.yaml).

Both UAMIs need DNS Zone Contributor on the public `jkslabs.site` DNS zone.
ExternalDNS also needs Reader on the DNS resource group for discovery. Assign
roles to UAMI principal IDs; ServiceAccounts and the issuer reference client IDs.
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
