# Azure Key Vault secrets

Shared Azure Helm chart for Key Vault connections and remote secret references.
Each cluster references this chart from its bootstrap Helmfile and supplies its own
values file. Install the External Secrets operator first. The AKS Envoy cluster
uses Helm release name `external-secrets-config`; keep that name when changing the
chart path so existing Helm ownership remains intact. Both releases stay
Helmfile-owned. These maps also own the optional `platform-secrets` store and
`argocd-server` ServiceAccount in the `argocd` namespace. Keep this chart
outside Argo CD cluster selection.

## Values maps

- `namespaces`: optional namespaces created before the namespaced resources. Namespaces are retained on uninstall. Disable an entry when the namespace already exists outside this release.
- `serviceAccounts`: Kubernetes accounts for vault access or a shared controller account. Map key becomes ServiceAccount name. Optional `automountServiceAccountToken` defaults to `false`; set it to `true` when a controller needs Kubernetes API access.
- `clusterSecretStores`: shared vault connections. Map key becomes ClusterSecretStore name; `spec` passes through unchanged.
- `externalSecrets`: optional remote secret references. Map key becomes ExternalSecret name; `spec` passes through unchanged.

Chart defaults use empty maps and render no resources. Define entries in each
cluster's `bootstrap/external-secrets-config/values.yaml`. AKS Envoy enables its platform namespace, reader, store, and OIDC ExternalSecret.
Generic workload examples start with `enabled: false`. Add entries or set maps to `{}`.
New entries default enabled when `enabled` omitted. Namespace defaults to release namespace for namespaced resources.
ClusterSecretStores have no namespace.

`ClusterSecretStore` connects to vault. `ExternalSecret` selects remote keys and creates Kubernetes Secret.
Actual password values stay in Key Vault. `remoteRef.key` holds remote secret name, not password.
No Kubernetes Secret containing password is rendered by Helm.

## Enable Azure example

1. Enable AKS OIDC issuer and workload identity in infrastructure.
2. Create Azure managed identity. Grant `Key Vault Secrets User` on required vault, or equivalent secret-read access policy.
3. Create federated credential using AKS OIDC issuer, audience `api://AzureADTokenExchange`,
   and subject `system:serviceaccount:external-secrets:eso-keyvault-reader`.
4. Fill reader client ID, tenant ID, and store vault URL in the cluster values file.
   Enable reader and store entries. ServiceAccount reference must match reader name and namespace.
5. Add allowed workload namespaces under store `spec.conditions`. Create namespaces separately or declare them in `namespaces`.
6. Create Key Vault secret `database-password`. Enable `externalSecrets.database-credentials`, or adapt remote key and target name.

Use referenced ServiceAccount authentication. Operator needs no Azure identity annotations or workload identity pod label.
This chart creates reader ServiceAccount; infrastructure owns Azure identity, federation, and vault permissions.
Allow operator network access to Azure identity endpoints and Key Vault.

To add another vault, copy store entry with unique map key. Add reader identity if permissions differ.
To sync more secrets, add `externalSecrets` entries or extend `spec.data` list.
Separate Helm ownership from any workload-owned ExternalSecrets; never manage same resource through both releases.

## Platform vault connection

- Namespaces: `argocd`, `cert-manager`, `external-dns`; bootstrap retains ownership.
- Reader ServiceAccount: `argocd-server`; store: `platform-secrets`.
- ExternalSecret: `argocd-oidc`; DNS controller client IDs stay in cluster ServiceAccount annotations.

| Configure | Value |
| --- | --- |
| Federation subject | `system:serviceaccount:argocd:argocd-server` |
| Federation issuer / audience | AKS OIDC URL / `api://AzureADTokenExchange` |
| Reader permissions | Key Vault secret-read access |
| Argo CD account | `server.serviceAccount.create: false`; token mounting enabled |
| Secret mapping | Key Vault `argocd-client-id` -> `argocd/argocd-oidc`, key `client-id` |
| Secret label | `app.kubernetes.io/part-of: argocd` |

- Terraform owns Azure identities, federation, and permissions.
- Reader UAMI and SSO application can have different client IDs; configure both federations.
- Argo CD reads SSO client ID through OIDC Secret reference; issuer stays in Helm values.
- Bootstrap waits for store and ExternalSecret readiness before installing Argo CD.
- Store currently omits `spec.conditions`; Kubernetes RBAC controls who can create consumers.
- Add namespace conditions when restricting access; current consumer: `argocd`.
- New remote key: add `externalSecrets` entry with target namespace; each namespace needs its own Secret.
- Namespace retention: `helm.sh/resource-policy: keep`. Disable bootstrap entry when another owner exists.
- Existing installation: retain resources in old release before transferring ownership. Never give Helm and Argo CD simultaneous ownership.
- Changing `server.serviceAccount.create` alone does not migrate ServiceAccount ownership.

## Render, review, deploy

From repository root, with existing bootstrap tools and target kubeconfig:

```sh
BOOTSTRAP=clusters/azure/DEV-JKS/dev/uks/sandbox/aks-envoy/bootstrap
CHART=catalog/charts/azure/key-vault-secrets
helm lint "$CHART" --strict --values "$BOOTSTRAP/external-secrets-config/values.yaml"
helm template external-secrets-config "$CHART" --namespace external-secrets \
  --values "$BOOTSTRAP/external-secrets-config/values.yaml"
# Export CILIUM_K8S_SERVICE_HOST as described in docs/helm-bootstrap.md.
helmfile --file "$BOOTSTRAP/helmfile.yaml" diff --suppress-secrets
# Review diff before confirming deployment.
helmfile --interactive --file "$BOOTSTRAP/helmfile.yaml" apply --suppress-secrets
```

Default render is empty until entries enabled. Helm readiness does not wait for store authentication or secret synchronization.
After enabling sample entries:

```sh
kubectl wait --for=condition=Ready clustersecretstore/azure-keyvault --timeout=120s
kubectl -n default wait --for=condition=Ready externalsecret/database-credentials --timeout=120s
kubectl -n default describe externalsecret database-credentials
```

Expect both resources Ready and target Secret `database-credentials` created in `default` namespace.
If authentication fails, check identity annotations, federation subject, vault permissions, and network access.
Keep secret values out of command output and Git.

References: [Azure provider](https://external-secrets.io/latest/provider/azure-key-vault/),
[ClusterSecretStore](https://external-secrets.io/latest/api/clustersecretstore/).
