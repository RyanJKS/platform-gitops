# ExternalDNS provider configuration

- Base Application: cloud agnostic; patch `__DNS_PROVIDER__` in cluster files.
- Copy provider values into cluster Application `spec.sources[0].helm.valuesObject`.
- Merge with shared values; replacing whole `valuesObject` requires retaining shared settings.
- Fill each `__...__` marker in `clusters/**/`. Keep catalog examples reusable.

| Provider | Copy or select | Cluster inputs |
| --- | --- | --- |
| Azure | [values.yaml](azure/values.yaml), [ConfigMap base](azure/kustomization.yaml) | Client ID, tenant, subscription, DNS resource group, zone, TXT owner |
| AWS | [values.yaml](aws/values.yaml) | IAM authentication, zones, TXT owner |
| GCP | [values.yaml](gcp/values.yaml) | Workload identity, project, zones, TXT owner |

Azure setup:

1. Copy Helm values into cluster patch; choose sources such as `ingress` or `gateway-httproute`.
2. Import `configs/azure/` from cluster Kustomization, or copy its ConfigMap locally.
3. Replace complete `data.azure.json`: tenant, subscription, resource group, `cloud`, `useWorkloadIdentityExtension`.
4. Set ServiceAccount client-ID and tenant annotations; retain workload identity pod label.
5. Match federation subject `system:serviceaccount:external-dns:external-dns`; audience `api://AzureADTokenExchange`.
6. Grant **DNS Zone Contributor** on zone and **Reader** on DNS resource group.

- ConfigMap wave `-5` requires namespace first.
- AKS Envoy bootstrap retains namespace ownership independently of ESO secrets.
- Other clusters: define `namespace.yaml` in cluster component folder at wave `-6` when no bootstrap owner exists.
- Child `CreateNamespace=true` runs too late for parent ConfigMap.
- AWS/GCP: omit Azure ConfigMap.
