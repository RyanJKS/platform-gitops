# Argo CD catalog

- [`application.yaml`](application.yaml): optional controller self-management Application.
- [`values.yaml`](values.yaml): cloud-neutral defaults shared by the controller Application and Helmfile.
- [`kustomization.yaml`](kustomization.yaml): selects the controller Application.
- [`configs/`](configs/README.md): provider copy templates, never loaded automatically.
- [Root configuration chart](../../charts/argocd/README.md): app-of-apps root, installed as a separate Helm release.

The controller Application and root chart serve different purposes. Importing this
catalog directory selects controller self-management; it does not install the root.
AKS Envoy keeps controller ownership in Helmfile. Its root source settings live in
`bootstrap/argocd-config/values.yaml`. Helmfile loads the shared controller values
first, then `bootstrap/argocd/values.yaml` for cluster overrides and SSO.
Other clusters can use the controller Application after the reviewed
[bootstrap ownership handoff](../../../docs/bootstrap.md).

The controller Application loads `values.yaml` through a values-only Git source.
Its default revision is `main`; cluster patches must select the revision containing
the shared file. AKS AGIC selects `dev/sandbox`, matching its root Application.
Inline `valuesObject` contains only cluster overrides and takes precedence over
the shared file. Helm merges nested maps, preserving shared health checks and CRDs.

Keep Azure identity, Entra OIDC, and AGIC settings under
[`configs/azure/`](configs/azure/README.md) as copy templates. Copy only the required
fields into cluster overrides and replace placeholders. Both existing clusters
override the catalog image tag with `""` to retain the pinned chart's `appVersion`.
