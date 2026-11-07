# AKS Envoy bootstrap

Terraform builds infrastructure. Helmfile installs bootstrap releases. Argo CD
then reconciles platform apps from Git.

## Installation flow

```mermaid
flowchart TD
    T["Terraform<br/>AKS BYOCNI, kube-proxy disabled<br/>OIDC, workload identity<br/>UAMIs, federation, Azure roles"]

    subgraph H["Helmfile"]
        C["1. Cilium<br/>CNI, Service routing, Hubble"]
        E["2. ESO<br/>Operator, webhook, cert controller"]
        P["3. Platform secrets<br/>argocd, cert-manager, external-dns namespaces<br/>Vault connection, Argo CD OIDC Secret"]
        A["4. Argo CD<br/>Controller, SSO, RBAC"]
        R["5. Root Application<br/>Git URL, revision, apps/ path"]
        C --> E --> P --> A --> R
    end

    subgraph GitOps["Argo CD: apps/platform/"]
        F["AppProject, OCI repository<br/>ExternalDNS ConfigMap<br/>Wave -5"]
        K["Gateway API CRDs<br/>Catalog Application, wave -4"]
        M["cert-manager<br/>Controller, CRDs, workload identity<br/>Wave -3"]
        EG["Envoy Gateway<br/>Controller + extension CRDs<br/>Creates destination namespace<br/>Wave -3"]
        D["ExternalDNS<br/>Controller + workload identity<br/>Wave -2"]
        W["GatewayClass, Gateway<br/>DNS-01 ClusterIssuer<br/>Argo CD HTTPRoutes"]
        F --> K --> EG
        F --> M --> EG
        EG --> D --> W
    end

    T --> C
    V["Key Vault<br/>argocd-client-id"] -.-> P
    R --> F
    W --> HTTPS["Public load balancer<br/>DNS record + Argo CD HTTPS"]
```

Numbers show bootstrap installation order. Helmfile waits for the store and ExternalSecret
`Ready`, then checks `argocd-oidc` exists before starting Argo CD. SSO reads
`client-id` from that Secret; issuer stays in [argocd/values.yaml](argocd/values.yaml).

cert-manager starts after Gateway CRDs and generates Certificate `argocd-server-tls`
from the annotated Gateway's HTTPS listener. Existing installations need the
[Certificate handoff](../../../../../../../../docs/helm-bootstrap.md#gateway-certificate-handoff)
before publishing this change; preserve the existing TLS Secret.

Root health checks gate child Applications between sync waves. Helmfile completion
does not mean DNS, certificates, or child Applications have finished reconciling.

## Prepare first

- Provision AKS with `network_plugin = "none"` and kube-proxy disabled. Enable
  OIDC and workload identity. Give the AKS identity required network permissions.
- Fill [platform secret values](external-secrets-config/values.yaml). Create Key
  Vault key `argocd-client-id` containing the Entra SSO application client ID.
- Configure Key Vault reader federation and secret-read access. Configure Entra
  federation, user assignments, and callback
  `https://argocd.jkslabs.site/auth/callback`. Both federations trust subject
  `system:serviceaccount:argocd:argocd-server`, the AKS OIDC issuer, and audience
  `api://AzureADTokenExchange`.
- Set dedicated ExternalDNS and cert-manager UAMI client IDs in cluster
  `apps/platform/<controller>/application.patch.yaml`; configure federation and DNS permissions.
  See [platform prerequisites](../apps/platform/README.md#terraform-inputs).

Platform secrets creates `argocd-server`; Argo CD uses `serviceAccount.create: false`.
The ServiceAccount name and namespace must match each federation's subject.
Azure credential display names do not need to match.

## Deploy locally

Run from repository root. Install Helm 3, Helmfile, helm-diff, kubectl, Azure CLI,
kubelogin, and Python 3. CI pins Helm `3.18.6` and Helmfile `1.1.7`.

```sh
az login
az aks get-credentials --resource-group YOUR_AKS_RESOURCE_GROUP --name YOUR_AKS_CLUSTER_NAME
kubelogin convert-kubeconfig --login azurecli
kubectl config current-context
kubectl cluster-info --request-timeout=30s

export CILIUM_K8S_SERVICE_HOST="$(
  kubectl config view --minify -o jsonpath='{.clusters[0].cluster.server}' |
    python3 -c 'import sys; from urllib.parse import urlsplit; print(urlsplit(sys.stdin.read().strip()).hostname or "")'
)"
: "${CILIUM_K8S_SERVICE_HOST:?Missing API hostname in current kubeconfig}"

BOOTSTRAP=clusters/azure/DEV-JKS/dev/uks/sandbox/aks-envoy/bootstrap/helmfile.yaml
helmfile --file "$BOOTSTRAP" diff --suppress-secrets
# Review the diff, then approve the interactive apply prompt.
helmfile --interactive --file "$BOOTSTRAP" apply --suppress-secrets
```

Use the same kubeconfig context for discovery and deployment. Re-export the host
after switching clusters. Set hostname only, without `https://` or port; AKS uses
port `443`. [cilium/api.yaml.gotmpl](cilium/api.yaml.gotmpl) passes this variable
into Cilium's `k8sServiceHost`. Missing variable stops Helmfile rendering.

From this `bootstrap/` directory, use `helmfile diff --suppress-secrets` and
`helmfile --interactive apply --suppress-secrets` after exporting the same API
hostname. Omit the repository-relative `--file` argument in that directory.

Gateway API CRDs are installed by the catalog Application after Argo CD starts,
before Envoy Gateway. Cilium's Gateway API support stays disabled, so bootstrap
has no dependency on these CRDs. The root applies CRDs directly with server-side
apply, avoiding Helm's release Secret size limit. No packaging helper is needed.
Before local bootstrap, run
`bash .github/scripts/check-gateway-api-ownership.sh` from the repository root.
Existing Helm owners require a reviewed migration; do not uninstall CRDs to
transfer ownership. CI checks for existing Helm ownership in both jobs.

Laptop kubeconfig authenticates Helmfile. Cilium pods do not inherit it. With
kube-proxy disabled, Cilium needs the direct API endpoint before Service routing
works. A successful `kubectl cluster-info` check does not supply that chart value.

## GitHub Actions and AKS runners

The current [workflow](../../../../../../../../.github/workflows/helm-bootstrap.yaml)
uses `runs-on: ubuntu-24.04`. Its
[setup action](../../../../../../../../.github/actions/helm-bootstrap-setup/action.yaml)
loads the target kubeconfig, checks API access, discovers the hostname, and exports
`CILIUM_K8S_SERVICE_HOST` in both plan and apply jobs. No manual export is needed.

An AKS-hosted self-hosted runner still needs this value. Runner location does not
configure Cilium's API connection. Keep discovery against the target cluster's
kubeconfig, even when the runner runs inside another AKS cluster.

Use an existing working runner for first bootstrap. An ordinary pod runner in
the fresh target cluster needs working CNI before it can start.

## Ownership and files

- [helmfile.yaml](helmfile.yaml): chart pins, release dependencies, readiness checks.
- [cilium/values.yaml](cilium/values.yaml): CNI and Hubble; Cilium Gateway controller disabled.
- [Shared ESO values](../../../../../../../../catalog/platform/external-secrets/values.yaml): generic operator settings.
- [external-secrets-config/values.yaml](external-secrets-config/values.yaml): target namespaces, shared ServiceAccount, Key Vault store, Argo CD OIDC ExternalSecret.
- [Shared Argo CD values](../../../../../../../../catalog/platform/argocd/values.yaml): generic defaults, health checks, CRDs, and default-deny RBAC.
- [Azure copy templates](../../../../../../../../catalog/platform/argocd/configs/azure/README.md): AGIC and Entra examples, never loaded automatically.
- [argocd/values.yaml](argocd/values.yaml): cluster installation, SSO, RBAC, and internal HTTP overrides, loaded after shared values.
- [argocd-config/values.yaml](argocd-config/values.yaml): root source, selecting [apps/](../apps/kustomization.yaml).

Helmfile owns these releases. Argo CD owns platform children, shared Gateway API
CRDs, and routing. The Gateway API catalog Application selects seven standard
CRDs at `v1.6.3`; Envoy's child installs only extension CRDs. Cilium preflight
stays disabled during initial installation.

Existing Helm or Terraform owners require an ownership handoff. Do not delete
CRDs or ServiceAccounts to transfer ownership. See the
[full guide](../../../../../../../../docs/helm-bootstrap.md) for migration, upgrades,
GitHub approval setup, and deployed-cluster checks.
