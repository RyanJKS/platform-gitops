# Platform GitOps

This repository defines Kubernetes platform capabilities and workload deployments for Argo CD.
Reusable definitions live in `catalog/`; cluster-specific intent lives in `clusters/`.
Infrastructure provisioning and container image builds are outside this repository.

- [Configuration boundaries](configuration-boundaries.md): shared defaults versus cluster implementation.
- [Architecture](architecture.md): repository boundaries, ownership, and current limitations.
- [Bootstrap](bootstrap.md): attach an existing cluster to Git.
- [Adding a cluster](adding-a-cluster.md): create an isolated cluster configuration.
- [Onboarding an application](onboarding-an-application.md): add a workload safely.
- [GitHub Actions runners](github-actions-runners.md): opt-in ARC controller and runner scale sets.
- [Promotion](promotion.md): promote reviewed configuration between environments.
- [Recovery](recovery.md): recover controllers and roll back configuration.

Azure dev and local kind configurations are provided. Azure prod and AWS dev remain reserved scaffolds.
No configuration in this repository creates a cloud cluster.
