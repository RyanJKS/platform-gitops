# Platform GitOps

Declarative Kubernetes platform capabilities and application deployments, reconciled by Argo CD.
Cluster provisioning, application source code, and image builds belong in other repositories.

## Layout

- `bootstrap/`: shared bootstrap instructions and examples.
- `clusters/.../bootstrap/root.yaml`: example roots; Terraform owns the Azure shared root.
- `catalog/platform/`: component inputs, an ApplicationSet template, matching permission bases, optional policy profiles, and shared Helm values.
- `catalog/applications/`: reusable workload bases.
- `clusters/`: explicit cloud, subscription/account alias, environment, region, spoke, and cluster configuration; only differences live here.
- `docs/`: architecture and operational guides.
- `scripts/validate.sh`: the same manifest checks used by CI.

Cluster paths identify their deployment target directly:

```text
clusters/azure/DEV-JKS/dev/eus2/spoke-atlas/aks-shared/
clusters/azure/DEV-JKS/dev/uks/spoke-atlas/aks-shared/
clusters/azure/DEV-JKS/dev/eus2/spoke-atlas/aks-atlas-market/
clusters/aws/personal-account/dev/eu-west-2/spoke-atlas/eks-shared/
clusters/local/kind/kind-platform/
```

Azure dev has shared (`chaos-generator`) and dedicated (`atlas-market`) cluster configurations.
Local kind has platform capabilities only. Azure prod and AWS dev are reserved scaffolds with no bootstrap roots.
The Azure shared cluster variants (`uks` and `eus2`) select only `chaos-generator`, with automatic sync under
Terraform-owned `all-apps` on `feature/repo-setup`. See [bootstrap](docs/bootstrap.md#azure-shared-chaos-generator-only).
Other cluster examples use manual sync. Other example workloads start at zero replicas with placeholder images and example DNS names.
These are starting configurations, not production-ready deployments.

## Start here

Follow [bootstrap](docs/bootstrap.md) to install Argo CD and attach exactly one cluster root.
Read [configuration boundaries](docs/configuration-boundaries.md) before adding shared defaults or cluster-specific behavior.
Use [application onboarding](docs/onboarding-an-application.md) to enable real workloads.

Optional [GitHub Actions runners](docs/github-actions-runners.md) provide ARC controller and scale-set
defaults with a disabled example; enable them only on selected clusters.

## Checks

Install Bash, kubectl with Kustomize, Helm, Mike Farah's yq v4, and kubeconform, then run:

```sh
scripts/validate.sh
```

Validation needs network access to chart repositories and Kubernetes schemas. CI pins tool versions in
[validate.yaml](.github/workflows/validate.yaml) and builds TechDocs using a container;
no Python application or Python development toolchain is maintained here.
See [CONTRIBUTING.md](CONTRIBUTING.md) for contribution and documentation checks.

Platform Applications are generated from explicit cluster selections. See
[adding a cluster or component](docs/adding-a-cluster.md) for examples and
[bootstrap migration](docs/bootstrap.md#migrate-existing-platform-applications) before
handing existing Applications to the ApplicationSet controller.
