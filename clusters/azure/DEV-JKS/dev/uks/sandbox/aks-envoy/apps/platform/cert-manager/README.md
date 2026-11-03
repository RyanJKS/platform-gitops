# cert-manager for AKS Envoy

This overlay imports the shared cert-manager Application and changes the
ClusterIssuer to production Azure DNS-01. It enables workload identity only on
the cert-manager controller. HTTP-01 and ingress-class configuration are absent
from the rendered issuer.

Store the UAMI client ID in Key Vault as `certmanager-client-id`. The central
[platform secrets configuration](../../../bootstrap/external-secrets-config/values.yaml)
creates the `cert-manager` namespace and an ExternalSecret targeting Kubernetes
Secret `certmanager-client-id`, key `client-id`. Both are owned by the bootstrap
`external-secrets-config` release. The controller reads `AZURE_CLIENT_ID` through
a Secret reference in [kustomization.yaml](kustomization.yaml). Kubernetes waits
for the Secret before starting the controller container.

The ServiceAccount keeps its tenant annotation but has no client-ID annotation.
The workload identity webhook injects the federated token and preserves the
explicit `AZURE_CLIENT_ID` reference. The ServiceAccount name and namespace must
match federated subject `system:serviceaccount:cert-manager:cert-manager`.
Grant this UAMI DNS Zone Contributor on the public `jkslabs.site` zone.
Review the issuer email, subscription, DNS resource group, and hosted zone.

[issuer-patches.yaml](issuer-patches.yaml) replaces the complete `/spec/acme`
mapping with the production DNS-01 configuration. It omits `managedIdentity`
so cert-manager uses the controller's ambient workload credentials. Keep the
ACME endpoint, email, account key Secret name, and all solvers in that mapping.
The controller explicitly enables cluster-issuer ambient credentials.

After changing the vault client ID, wait for the ExternalSecret to become Ready
and restart the cert-manager Deployment; environment variables do not refresh
inside running containers. Ensure the `platform-secrets` reader can read the
vault key and the store allows the `cert-manager` namespace.

No Azure client secret or Gateway API feature flag is required for DNS-01.
The Argo CD Certificate lives in [../argocd/certificate.yaml](../argocd/certificate.yaml).
See [platform configuration](../README.md) for dependencies and ExternalDNS's
separate identity and permissions.
