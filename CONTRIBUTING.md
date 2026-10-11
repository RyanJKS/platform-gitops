# Contributing

1. Create a branch from `main` and keep changes focused.
2. Render affected cluster Kustomizations and review their resources.
3. For Argo CD installation changes, lint and render the pinned official Helm chart with shared, cloud, and cluster values.
4. Update documentation when behavior, configuration, or operations change.
5. Open a pull request describing the problem, resulting behavior, and validation.

Local rendering and Helm lint check configuration. They do not contact the
Kubernetes API or prove readiness, cloud permissions, DNS, or live resource
ownership.

For installation commands and value ordering, follow [bootstrap](docs/bootstrap.md).
For example, render a managed cluster root from the repository root:

```sh
kubectl kustomize clusters/azure/DEV-JKS/dev/uks/sandbox/aks-agic
```

Run the strict documentation build:

```sh
docker run --rm -v "$PWD:/content:ro" -w /content spotify/techdocs:v1.2.6 \
  build --strict --site-dir /tmp/site
```

Keep `mkdocs.yml`, `docs/`, and the `dir:.` TechDocs annotation in
`catalog-info.yaml` aligned. Do not commit generated documentation, credentials,
kubeconfig files, or infrastructure state.

## Configuration placement

The [platform catalog](docs/platform-catalog.md) owns reusable definitions and
shared defaults. Cluster files select components and override identities,
permissions, namespaces, domains, storage, and operational sizing. Review all
selected consumers when changing a shared default.

Argo CD installation values are consumed by its Helm or Terraform owner. Child
Application values are consumed by Argo CD. Keep one active owner for each
installation. Chart upgrades require review of upstream compatibility, CRD
changes, selectors, and migration notes. Dependabot maintains GitHub Actions;
chart and image pins receive deliberate review.
