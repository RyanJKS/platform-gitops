# Argo CD catalog

- [`application.yaml`](application.yaml): controller Application for clusters that manage Argo CD through Argo CD.
- [`kustomization.yaml`](kustomization.yaml): selects that controller Application.
- [`configs/`](configs/README.md): provider and SSO examples.
- [Root configuration chart](../../charts/argocd/README.md): reusable app-of-apps root, installed as a separate Helm release.

The controller Application and root chart serve different purposes. Importing
this directory selects controller self-management; it does not install the root.
AKS Envoy instead uses the official controller Helm chart plus the shared root
chart in Helmfile. Cluster root source settings live in
`bootstrap/argocd-config/values.yaml`; controller and SSO settings stay in
`bootstrap/argocd/values.yaml`.

Other clusters can use the controller Application and provider profiles after a
reviewed ownership handoff. See [bootstrap ownership handoff](../../../docs/bootstrap.md).
