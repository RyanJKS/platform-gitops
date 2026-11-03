# ExternalDNS provider configuration

Provider values are templates for cluster Application patches. Keep cloud
identities, DNS filters, and TXT owner IDs in the cluster overlay.

The `azure/` Kustomization is a reusable ConfigMap base for the `external-dns`
namespace. Import it from the cluster's ExternalDNS Kustomization and patch
`data.azure.json` with the tenant, subscription, and DNS resource group. This
field is a JSON string, so the patch replaces the complete JSON document,
including `cloud` and `useWorkloadIdentityExtension`.

The ConfigMap uses sync wave `-5`. Its namespace must exist before the parent
applies it. AKS Envoy's bootstrap secrets release creates `external-dns` for its
identity Secret. Other clusters can select
`catalog/platform/namespaces/external-dns/` at wave `-6` if the namespace has no
bootstrap owner. The controller's `CreateNamespace=true` runs only during child
sync and cannot satisfy an earlier parent ConfigMap.

AWS and GCP configurations do not use the Azure ConfigMap.
