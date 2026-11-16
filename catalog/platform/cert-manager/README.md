# cert-manager

- Base: cloud-agnostic Application, CRDs, API readiness hooks, staging HTTP-01 issuer.
- Default issuer: set `__INGRESS_CLASS_NAME__` through cluster patches.
- Azure DNS-01: copy examples into `clusters/**/`; fill each `__...__` marker there.

| Example | Copy into cluster |
| --- | --- |
| [Azure Helm values](configs/azure/values.yaml) | Application `spec.sources[0].helm.valuesObject`; merge with shared values |
| [Azure issuer patch](configs/azure/issuer.patch.yaml) | Local `issuer.patch.yaml`; target `ClusterIssuer/letsencrypt` |

- Patch individual Helm fields to preserve shared CRDs and API readiness hooks.
- Replacing all `valuesObject` requires copying shared values too.
- Identity: ServiceAccount annotations + workload identity pod label.
- Federation subject: `system:serviceaccount:cert-manager:cert-manager`; audience: `api://AzureADTokenExchange`.
- Grant identity **DNS Zone Contributor** on selected zone.
- DNS-01 uses ambient controller credentials; keep `--cluster-issuer-ambient-credentials=true`.
- Issuer patch replaces complete `/spec/acme`; retain endpoint, email, account key, and solvers.
- Account key stays in cert-manager's generated Kubernetes Secret.
