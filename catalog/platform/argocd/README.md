# Argo CD catalog

- `bootstrap/`: reusable root manifest templates and README; excluded from Kustomize.
- `root_project.yaml`: GitOps-owned self-management AppProject at wave `-4`.
- `application.yaml`: GitOps self-management child and runtime chart metadata.
- `kustomization.yaml`: selects the project and self-management Application.
- `values.yaml`: shared runtime Helm values.
- `azure-values.yaml`: separate Azure AGIC runtime profile.

Terraform installs a hardcoded chart with chart defaults and creates `all-apps`
in `default`, supplying the cluster path and Git revision. It does not create
the self-management project or read runtime values. The cluster root imports this
catalog and creates the `bootstrap` project before its Argo CD child. Catalog
templates contain no cluster paths.

The intended GitOps child uses the existing `bootstrap` project and applies shared,
Azure, and cluster values after Helm ownership is relinquished. The cluster's
automatic-sync gate remains disabled during that handoff.

See [bootstrap and ownership handoff](../../../docs/bootstrap.md).
