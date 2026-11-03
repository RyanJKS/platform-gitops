# Setup

Use ESO (External Secrets Operator) -> https://external-secrets.io/latest/introduction/overview/

Create `SecretStore` to "mount" to keyvault and then `ExternalSecret` so it can be pulled similar to `Secret`

**`SecretStore` is usable within one namespace. `ClusterSecretStore` can be shared across namespaces.** Both describe how ESO connects and authenticates to Key Vault; neither contains the actual secret values.
Docs: https://external-secrets.io/latest/api/secretstore/

AKS Envoy uses Helmfile for operator and shared Azure configuration chart.
See [configuration chart](../../charts/azure/key-vault-secrets/README.md)
for map-based ClusterSecretStores, reader ServiceAccounts, and ExternalSecrets.
Keep catalog Argo CD Application unselected when Helmfile owns operator.

## Shared operator values

[`values.yaml`](values.yaml) contains provider-neutral operator settings, including
ESO CRDs, ServiceAccount, webhook, and its certificate controller. Helmfiles can
reference this file directly and add a cluster values file afterward only when
an override is needed. AKS Envoy reads these values from the catalog; it keeps
vault connections and remote secret references in its cluster configuration.

`application.yaml` remains an alternative Argo CD-managed installation using the
official chart defaults and its inline `installCRDs` setting. It does not consume
this values file automatically. Do not select that Application when Helmfile
owns the operator.

## Central platform Secrets

Declare shared vault connections and ExternalSecrets in the cluster's
`bootstrap/external-secrets-config/values.yaml`. The reusable
`catalog/charts/azure/key-vault-secrets` chart creates the selected namespaces
and ExternalSecrets. No per-controller identity directory is required.

AKS Envoy maps Key Vault key `certmanager-client-id` to a Secret of the same name
in `cert-manager`, and `externaldns-client-id` to a Secret of the same name in
`external-dns`. Both Secrets contain key `client-id`. Controllers consume them
through `AZURE_CLIENT_ID` with a `secretKeyRef`, retaining the workload identity
pod label and tenant annotation. The Azure webhook supplies the projected token
and preserves the explicit environment variable. cert-manager's issuer uses
ambient workload credentials instead of an explicit `managedIdentity.clientID`.

Secret references are namespace-scoped. To expose another key to a workload,
add an `externalSecrets` entry with its target namespace, Secret name, and key,
then reference that Secret from the consumer in the same namespace. To expose
the same value in multiple namespaces, declare a target in each namespace.
The shared store must allow those namespaces and its reader needs access to the
remote keys. Client IDs are public identifiers; this centralizes configuration
without adding client secrets.

Keep the bootstrap reader identity configured independently so ESO can establish
the connection first. Restart consuming Pods after environment-backed Secrets
change; existing environment variables do not refresh automatically.

|                       | `SecretStore`                                        | `ClusterSecretStore`                                                             |
| --------------------- | ---------------------------------------------------- | -------------------------------------------------------------------------------- |
| Kubernetes scope      | Namespaced                                           | Cluster-wide                                                                     |
| `metadata.namespace`  | Required for your intended namespace                 | Omitted                                                                          |
| Who can reference it? | `ExternalSecret` resources in the **same namespace** | `ExternalSecret` resources across namespaces, subject to configured restrictions |
| Typical use           | Separate access for each application or team         | A centrally managed connection shared by several namespaces                      |
| Reference kind        | `kind: SecretStore`                                  | `kind: ClusterSecretStore`                                                       |
