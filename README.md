# Platform GitOps

Declarative Kubernetes platform capabilities and application deployments, reconciled by Argo CD.
Cluster provisioning, application source code, and image builds belong in other repositories.

## Layout

- `bootstrap/roots/`: one root Application per configured cluster.
- `catalog/platform/`: shared controller defaults, namespace identities, Helm values, and gateway controller resources.
- `catalog/applications/`: reusable workload bases.
- `clusters/`: explicit cloud, subscription/account alias, environment, region, spoke, and cluster configuration; only differences live here.
- `docs/`: architecture and operational guides.
- `scripts/validate.sh`: the same manifest checks used by CI.

Cluster paths identify their deployment target directly:

```text
clusters/azure/DEV-JKS/dev/uksouth/spoke-atlas/aks-shared/
clusters/azure/DEV-JKS/dev/uksouth/spoke-atlas/aks-atlas-market/
clusters/azure/DEV-JKS/prod/uksouth/spoke-atlas/aks-shared/
clusters/aws/personal-account/dev/eu-west-2/spoke-atlas/eks-shared/
clusters/local/kind/kind-platform/
```

Azure dev has shared (`atlas-ml`) and dedicated (`atlas-market`) cluster configurations.
Local kind has platform capabilities only. Azure prod and AWS dev are reserved scaffolds with no bootstrap roots.
All syncs are manual. Workloads start at zero replicas with placeholder images and example DNS names.
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
