# Bootstrap templates

These reusable manifests are excluded from the catalog Kustomization. They do
not select a cluster.

- `root_project.yaml`: bootstrap AppProject template.
- `root_application.yaml`: root Application template. Set its
  `spec.source.path` (`REPLACE_PLACEHOLDER`), repository URL, and Git revision in
  the cluster/bootstrap consumer before deploying it.

Install Argo CD first, then create the project, then the root Application. The
root selects the cluster Kustomization, which imports catalog Applications and
applies cluster patches. Child Applications deploy charts and their own config
or workload overlays.

Current Terragrunt bootstrap reads `catalog/platform/argocd/root_project.yaml`
and defines the root Application inline. It does not consume these nested
bootstrap templates automatically. See [bootstrap and ownership handoff](../../../../docs/bootstrap.md).
