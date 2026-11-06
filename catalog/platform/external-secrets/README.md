# Setup

Use ESO (External Secrets Operator) -> https://external-secrets.io/latest/introduction/overview/

Create `SecretStore` to "mount" to keyvault and then `ExternalSecret` so it can be pulled similar to `Secret`

**`SecretStore` is usable within one namespace. `ClusterSecretStore` can be shared across namespaces.** Both describe how ESO connects and authenticates to Key Vault; neither contains the actual secret values.
Docs: https://external-secrets.io/latest/api/secretstore/

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
