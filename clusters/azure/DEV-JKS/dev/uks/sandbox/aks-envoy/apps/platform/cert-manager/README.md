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

Reusable examples: [catalog](../../../../../../../../../../catalog/platform/cert-manager/README.md).
Dependencies and permissions: [platform configuration](../README.md).
Certificate: [Argo CD](../argocd/certificate.yaml).
