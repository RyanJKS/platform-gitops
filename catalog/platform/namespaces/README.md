# Platform namespaces

Each subdirectory is an independently selectable Kustomize base. Cluster
`apps/platform/namespaces/kustomization.yaml` files import only the namespaces
required by their selected controllers. There is no aggregate catalog
Kustomization that creates every namespace.

Namespaces use Argo CD sync wave `-6`, before root-managed configuration at
wave `-5` and controller Applications. Import the namespace selection into the
same root as those resources; the folder name alone does not establish order.
Patch cluster-specific namespace labels and annotations in the cluster overlay.

AKS Envoy selects only `envoy-gateway-system` through its Argo CD root. Its
`external-secrets-config` bootstrap release creates `argocd`, `cert-manager`, and
`external-dns` before placing platform ExternalSecrets there. Other bootstrap
namespaces also retain their existing owners;
do not select those namespaces in Argo CD as well. Keep one owner per namespace.
Child Applications retain `CreateNamespace=true` for selections that do not use
this layer. Changes to ownership of existing namespaces require review before
syncing.
