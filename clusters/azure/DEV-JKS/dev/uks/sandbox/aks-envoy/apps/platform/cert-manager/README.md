# cert-manager for AKS Envoy

- Shared Application + production Azure DNS-01 issuer.
- Bootstrap owns namespace; Argo CD owns controller and issuer.

| File | Configure |
| --- | --- |
| [application.patch.yaml](application.patch.yaml) | Replace `__CERT_MANAGER_CLIENT_ID__` with dedicated UAMI client ID; review tenant |
| [issuer.patch.yaml](issuer.patch.yaml) | ACME email, subscription, DNS resource group, hosted zone |
| [kustomization.yaml](kustomization.yaml) | Catalog base + patch targets |

- ServiceAccount: `cert-manager`; federation subject: `system:serviceaccount:cert-manager:cert-manager`.
- Audience: `api://AzureADTokenExchange`; issuer: cluster AKS OIDC URL.
- Grant **DNS Zone Contributor** on `jkslabs.site`.
- Workload identity webhook injects Azure environment variables from ServiceAccount annotations.
- Issuer uses ambient controller credentials; retain `--cluster-issuer-ambient-credentials=true`.
- `/spec/acme` patch replaces whole mapping; retain endpoint, email, account key, and solvers.
- After identity change: sync Application, verify ServiceAccount annotation, restart controller Pods.

Gateway certificate creation is enabled with `config.enableGatewayAPI: true`.
The cluster moves cert-manager to root wave `-3`, after Gateway API CRDs at `-4`,
so Gateway discovery works at controller startup. Envoy Gateway also uses wave `-3`.
The [Gateway](../gateways/gateway.yaml) selects `letsencrypt-prod` through its
`cert-manager.io/cluster-issuer` annotation. cert-manager creates Certificate
`argocd-server-tls` and its TLS Secret in namespace `argocd` from the HTTPS listener.
No separate Certificate manifest is required.

Existing installations need the [Certificate ownership handoff](../../../../../../../../../../docs/helm-bootstrap.md#gateway-certificate-handoff)
before publishing the updated Gateway configuration. Keep the TLS Secret during handoff.

Reusable examples: [catalog](../../../../../../../../../../catalog/platform/cert-manager/README.md).
Dependencies and permissions: [platform configuration](../README.md).
