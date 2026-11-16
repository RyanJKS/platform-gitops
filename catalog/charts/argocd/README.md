# Argo CD root configuration

This chart creates only the root app-of-apps Application. The official `argo-cd`
chart installs the controller in a separate release. SSO, RBAC, and controller
settings do not belong in this root chart.

The defaults are `all-apps`, project `default`, local cluster destination,
automatic sync with self-healing, pruning disabled, and server-side apply.
The root namespace is the Helm release namespace. It has no Helm hooks or
cascading-deletion finalizer.

Each cluster supplies `source.repoURL`, `source.targetRevision`, and `source.path`
in its `argocd-config/values.yaml`. Rendering fails if repository URL or path is
empty. Other defaults can be overridden in the same values file.

AKS Envoy installs release `argocd-config` after `argocd` and waits for the
Application CRD to be Established. A fresh-install diff disables API validation
for the root release because Argo CD has not installed that CRD yet.

From the repository root:

```sh
BOOTSTRAP=clusters/azure/DEV-JKS/dev/uks/sandbox/aks-envoy/bootstrap
helm lint catalog/charts/argocd --strict --values "$BOOTSTRAP/argocd-config/values.yaml"
helm template argocd-config catalog/charts/argocd --namespace argocd \
  --values "$BOOTSTRAP/argocd-config/values.yaml"
```

Root and controller remain separate Helm releases. Updating controller values
does not replace the root. Uninstalling `argocd-config` removes the root; without
a cascading-deletion finalizer, child Applications remain in place. An existing
root created outside this release needs a reviewed Helm ownership migration
before adoption. Do not let two tools manage the same root.
