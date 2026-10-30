# Setup

Use ESO (External Secrets Operator) -> https://external-secrets.io/latest/introduction/overview/

Create `SecretStore` to "mount" to keyvault and then `ExternalSecret` so it can be pulled similar to `Secret`

**`SecretStore` is usable within one namespace. `ClusterSecretStore` can be shared across namespaces.** Both describe how ESO connects and authenticates to Key Vault; neither contains the actual secret values.
Docs: https://external-secrets.io/latest/api/secretstore/

AKS Envoy uses Helmfile for operator and separate local configuration chart.
See [configuration chart](../../../clusters/azure/DEV-JKS/dev/uks/sandbox/aks-envoy/bootstrap/external-secrets-config/README.md)
for map-based ClusterSecretStores, reader ServiceAccounts, and ExternalSecrets.
Keep catalog Argo CD Application unselected when Helmfile owns operator.

|                       | `SecretStore`                                        | `ClusterSecretStore`                                                             |
| --------------------- | ---------------------------------------------------- | -------------------------------------------------------------------------------- |
| Kubernetes scope      | Namespaced                                           | Cluster-wide                                                                     |
| `metadata.namespace`  | Required for your intended namespace                 | Omitted                                                                          |
| Who can reference it? | `ExternalSecret` resources in the **same namespace** | `ExternalSecret` resources across namespaces, subject to configured restrictions |
| Typical use           | Separate access for each application or team         | A centrally managed connection shared by several namespaces                      |
| Reference kind        | `kind: SecretStore`                                  | `kind: ClusterSecretStore`                                                       |
