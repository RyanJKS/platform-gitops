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

AKS Envoy enables `namespaces.argocd`, `serviceAccounts.argocd-server`,
`clusterSecretStores.platform-secrets`, and `externalSecrets.argocd-oidc`.
Configure the reader client ID, tenant ID, and vault URL before applying. Create
Azure federation for subject
`system:serviceaccount:argocd:argocd-server`, with the AKS OIDC issuer and
audience `api://AzureADTokenExchange`, and grant the identity vault secret-read
access. Terraform owns these Azure resources.

This chart creates the `argocd` namespace, shared `argocd-server` ServiceAccount,
and platform secrets before Argo CD is installed. Argo CD sets
`server.serviceAccount.create: false` and uses that account. The platform store
references it for Key Vault access. The account name and namespace must match
the UAMI federated credential subject `system:serviceaccount:argocd:argocd-server`;
the credential's display name can differ. The shared account enables token
mounting for the server's Kubernetes API access. It depends only on the External Secrets operator. Argo CD then waits
for this store and ExternalSecret to be Ready and checks that the Secret exists.
The namespace has `helm.sh/resource-policy: keep` so removing the secrets release
does not delete the namespace containing Argo CD. If the namespace already
exists outside this release, disable `namespaces.argocd` to avoid taking ownership.

The store allows only `argocd` by default. Extend
`clusterSecretStores.platform-secrets.spec.conditions` when needed.
`argocd-oidc` maps remote keys `argocd-oidc-issuer` and `argocd-oidc-client-id`
to `issuer` and `client-id` in a Secret labeled
`app.kubernetes.io/part-of: argocd`. The issuer is the complete Entra issuer URL;
the client ID is the SSO app registration GUID. Argo CD reads these keys through
its OIDC Secret references. The UAMI annotations on this account select the
identity used by ESO. The SSO app has its own federation for the same Kubernetes
subject and can have a different client ID; Argo CD reads its SSO client ID from
the Secret. Workload identity needs no client secret.

For an existing installation, retain the ServiceAccount in the old Helm release
before removing it and transferring ownership to `external-secrets-config`.
Changing `server.serviceAccount.create` alone does not migrate ownership.

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
