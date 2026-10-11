# Platform GitOps

Declarative Kubernetes platform capabilities and application deployments,
reconciled by Argo CD. Cluster provisioning, application source code, and image
builds belong in other repositories.

- `catalog/platform/`: reusable controller Applications, Helm values, and configuration.
- `catalog/applications/`: reusable workload manifests.
- `clusters/`: cluster selections, permissions, and deployment-specific overrides.
- `docs/`: bootstrap, configuration, and debugging guides.

Cloud paths use
`clusters/<cloud>/<account>/<environment>/<region>/<domain>/<cluster>/`.

Terraform installs the official Argo CD chart with minimal chart defaults and
creates the root Application. The root creates the AppProjects from Git. Shared catalog, Azure, and
cluster values configure the GitOps self-management child after handoff.
The AKS AGIC root then creates an Argo CD self-management child. Its automatic
sync stays disabled until the documented ownership handoff completes.

For the UK South AGIC sandbox, follow [bootstrap and handoff](docs/bootstrap.md).
Projects and children use sync waves `-5` through `1`. After runtime values are
applied, the root waits for each child to report `Synced` and `Healthy`. Minimal
bootstrap lacks that health customization; verify initial children manually.
Verify Azure access and routing separately. Pruning remains disabled.

Render the child definitions from the repository root:

```sh
kubectl kustomize clusters/azure/DEV-JKS/dev/uks/sandbox/aks-agic
```

Read [platform catalog selections](docs/platform-catalog.md) and
[debugging](docs/debugging.md) before changing a live deployment. Terraform retains
ownership of existing Azure extension-managed Argo CD installations.

Build documentation with the pinned TechDocs image:

```sh
docker run --rm -v "$PWD:/content:ro" -w /content spotify/techdocs:v1.2.6 \
  build --strict --site-dir /tmp/site
```

Do not commit generated documentation, credentials, or infrastructure state.
