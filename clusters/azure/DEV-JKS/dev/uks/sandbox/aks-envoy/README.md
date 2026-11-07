# AKS Envoy sandbox

Terraform provisions AKS BYOCNI, OIDC, workload identity, Azure identities, and
role assignments. The [bootstrap workflow](../../../../../../../.github/workflows/helm-bootstrap.yaml)
then installs Cilium, ESO, platform secrets, Argo CD with
SSO, and the separate root Application release. See the
[bootstrap README](bootstrap/README.md) for the dependency diagram and ownership.

The root selects [apps/platform](apps/platform/README.md) on `feature/monitoring`:
Gateway API CRDs, cert-manager, Envoy Gateway, ExternalDNS, the platform AppProject, and Argo CD
HTTPS routing. Commit and push these files before deploying. Keep the root and
Envoy values Git source on the same revision. Private Git repository credentials
must be configured separately.

## Configure before deployment

- Provision AKS with `network_plugin = "none"`, OIDC, and workload identity.
  The bootstrap workflow rejects other network plugins. Terraform must supply
  the control-plane identity's required network permissions for load balancers.
- Review the Cilium pod pool `10.244.0.0/16` against Azure network ranges.
  Cilium handles the CNI and Hubble; its Gateway API controller is disabled.
- Fill the Key Vault identity and vault settings in
  `bootstrap/external-secrets-config/values.yaml`, and configure Entra SSO
  federation and the callback URI.
- Create dedicated ExternalDNS and cert-manager UAMIs, federation, and DNS roles.
  Fill `__EXTERNAL_DNS_CLIENT_ID__` and `__CERT_MANAGER_CLIENT_ID__` in
  `apps/platform/<controller>/application.patch.yaml` ServiceAccount annotations.
  Envoy Gateway needs no workload UAMI; AKS control-plane identity handles network resources.
- Review `jkslabs.site`, the Azure DNS resource group and subscription, and
  existing hostname ownership before syncing. cert-manager uses Azure DNS-01.

Argo CD's controller configuration stays in `bootstrap/argocd/values.yaml`.
The root's source values stay in `bootstrap/argocd-config/values.yaml`.
Envoy terminates public HTTPS and forwards to Argo CD's internal HTTP Service.
The ServiceAccount `argocd-server` remains owned by platform secrets.

## Ownership and verification

Helmfile keeps ownership of Cilium, ESO, platform secrets, Argo CD, and the root.
Argo CD owns shared Gateway API CRDs, selected platform controllers, and routing.
Do not import bootstrap releases into the root, or
run infrastructure Argo CD/root Helm units against this cluster.

Helmfile readiness does not wait for the root's children, DNS propagation,
certificate issuance, or SSO login. Follow the full
[deployment and verification guide](../../../../../../../docs/helm-bootstrap.md),
including GitHub environment configuration and existing ownership migrations.
