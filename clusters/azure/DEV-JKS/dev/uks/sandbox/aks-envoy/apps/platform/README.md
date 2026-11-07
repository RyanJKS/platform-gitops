# AKS Envoy platform applications

| Root wave | Resources |
| --- | --- |
| `-5` | AppProject, OCI repository, ExternalDNS ConfigMap |
| `-4` | Gateway API CRDs, cert-manager |
| `-3` | Envoy Gateway |
| `-2` | ExternalDNS |
| `0` | GatewayClass, Gateway, ClusterIssuer, Certificate, HTTPRoutes |

- Child health checks gate later waves; routing and certificates converge at wave `0`.
- Bootstrap retains `argocd`, `cert-manager`, `external-dns`, `external-secrets`, and `kube-system` ownership.
- Envoy Application creates `envoy-gateway-system` during sync with `CreateNamespace=true`.
- Patch files: `<resource>.patch.yaml`; `kustomization.yaml` selects targets.
- Catalog holds reusable code; `configs/<cloud>/` holds provider examples; cluster files hold deployment values.

## Terraform inputs

| Controller | Client ID field | Federated subject |
| --- | --- | --- |
| cert-manager | [application.patch.yaml](cert-manager/application.patch.yaml): `__CERT_MANAGER_CLIENT_ID__` | `system:serviceaccount:cert-manager:cert-manager` |
| ExternalDNS | [application.patch.yaml](external-dns/application.patch.yaml): `__EXTERNAL_DNS_CLIENT_ID__` | `system:serviceaccount:external-dns:external-dns` |

- Replace both placeholders before sync. Client IDs belong in ServiceAccount annotations.
- Federation: AKS OIDC issuer + audience `api://AzureADTokenExchange`.
- Both UAMIs: **DNS Zone Contributor** on `jkslabs.site`.
- ExternalDNS also needs **Reader** on DNS resource group.
- Azure role assignments use principal IDs; annotations use client IDs.
- After identity change: sync Application, verify annotation, restart controller Pods.
- AKS control-plane identity handles load balancers and networking; Envoy Gateway needs no workload UAMI here.

## DNS and HTTPS

- Azure tenant, subscription, resource group: [configmap.patch.yaml](external-dns/configmap.patch.yaml).
- ExternalDNS: `gateway-httproute`, `upsert-only`, owner `dev-uks-sandbox-aks-envoy`.
- Review existing record ownership before switching clusters.
- cert-manager: production DNS-01; dedicated identity writes challenge TXT records.
- HTTPS: `argocd.jkslabs.site` -> Gateway TLS -> `argocd-server:80`.
- Certificate: `argocd-server`; TLS Secret: `argocd-server-tls`; HTTP redirects to HTTPS.
- Listeners accept routes only from `argocd`. Argo CD CLI requires `--grpc-web`.
- Argo CD SSO and ServiceAccount stay in bootstrap.
- Keep root and Envoy values Git revisions aligned: currently `feature/monitoring`.

## Verify

From repository root:

```sh
kubectl kustomize clusters/azure/DEV-JKS/dev/uks/sandbox/aks-envoy/apps
```

[Deployment guide](../../../../../../../../../docs/helm-bootstrap.md#platform-applications-and-azure-identities): Terraform handoff, sync order, runtime checks.
