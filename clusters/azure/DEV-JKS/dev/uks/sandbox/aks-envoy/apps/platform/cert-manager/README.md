# cert-manager for AKS Envoy

This overlay imports the shared cert-manager Application and changes the
ClusterIssuer to production Azure DNS-01. It enables workload identity only on
the cert-manager controller. HTTP-01 and ingress-class configuration are absent
from the rendered issuer.

Set `REPLACE_CERT_MANAGER_UAMI_CLIENT_ID` to the same Terraform output in
[kustomization.yaml](kustomization.yaml) and
[issuer-patches.yaml](issuer-patches.yaml). The ServiceAccount name and namespace
must match federated subject `system:serviceaccount:cert-manager:cert-manager`.
Grant this UAMI DNS Zone Contributor on the public `jkslabs.site` zone.

Review the issuer email, subscription, DNS resource group, and hosted zone.
No Azure client secret or Gateway API feature flag is required for DNS-01.
The Argo CD Certificate lives in [../argocd/certificate.yaml](../argocd/certificate.yaml).
See [platform configuration](../README.md) for dependencies and ExternalDNS's
separate identity and permissions.
