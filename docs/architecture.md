# Architecture

## Repository boundaries

`bootstrap/roots/` maps each configured cluster to its Argo CD Kustomization. That Kustomization selects
shared controller Applications and cluster-owned AppProjects, resources, and workloads. Each child
Application reads either a pinned Helm chart with values files or a Kustomize path.

`catalog/` holds reusable definitions. `clusters/` expresses cloud, subscription/account alias, environment, region, spoke, cluster,
and then `argocd/`, `platform/`, or `applications/`. Local paths omit environment, account, and region segments.
Azure uses a subscription alias; AWS uses an account alias. The initial aliases are
`DEV-JKS` and `personal-account`. Map aliases to actual cloud IDs in infrastructure
configuration; these folders identify deployment targets and do not enforce access isolation.

Read [configuration boundaries](configuration-boundaries.md) before deciding where a new resource belongs.

## Shared installation defaults

- `catalog/platform/defaults/argocd/` contains five Applications for Argo CD, cert-manager,
  External Secrets, monitoring, and Envoy Gateway. It contains no AppProject or Gateway Application.
- `catalog/platform/namespaces/base/` defines the five platform namespace identities once.
  Application namespace identities live in `catalog/applications/<application>/namespace/`.
- `catalog/platform/gateway/resources/` defines the reusable Envoy GatewayClass only.
- Shared Helm values select portable installation behavior, such as CRD installation, ClusterIP access
  for Argo CD, and the Kubernetes provider for Envoy. Grafana is disabled in the starter monitoring setup.
- Workload bases define starter containers, resource requests, security contexts, services, and zero
  replicas. These are application defaults that cluster overlays can replace, not cluster-wide policy.

Clusters explicitly import the controller bundle. Use Kustomize deletion patches to omit capabilities
and keep namespace selection aligned. No ApplicationSet or additional templating layer is required.

## Cluster decisions

Each cluster owns its platform and workload AppProjects, namespace composition and security labels,
quota amounts, Gateway listeners and route permissions, monitoring retention and sizing, and controller
replica counts. Local Applications point to these resources. The Gateway Kustomization imports the shared
GatewayClass and adds its own Gateway. Helm values are appended through the cluster's Argo CD patches.

The supplied settings preserve the starter behavior: restricted Pod Security Admission at `v1.30`,
workload namespace quotas, HTTP-only Gateways permitting labeled namespaces, one gateway controller
replica, and three-day Prometheus retention. Kind requests less Prometheus memory than Azure dev.
These settings are independent cluster choices, even when the initial values match.

## Configured clusters

| Cluster | Platform | Workloads |
| --- | --- | --- |
| Azure dev `aks-shared` | Shared controllers and local configuration | Atlas ML inference |
| Azure dev `aks-atlas-market` | Shared controllers and local configuration | Atlas Market API and worker |
| Local `kind-platform` | Shared controllers and local configuration | None |
| Azure prod `aks-shared` | Reserved only | None |
| AWS dev `eks-shared` | Reserved only | None |

Each configured cluster runs its own Argo CD. Child Application names repeat across clusters, so never
apply multiple roots to the same Argo CD namespace. Roots use the bootstrap `default` project.
Platform projects have broad starter administrator permissions; their cluster-local location makes
future refinement independent. Workload projects allow Deployments, Services, and HTTPRoutes in their
own namespace. Operator SSO/RBAC and cloud workload identity are not configured.

## Reconciliation and promotion

Helm Applications use a chart pinned to an exact version and this repository at `main` as `$values`.
Shared values load first; cluster values load last. Argo CD renders templates rather than creating
Helm releases. Shared catalog changes affect every consumer on its next relevant sync.
See [promotion](promotion.md) for staged changes and versioning.

Sync is manual and pruning is opt-in. Sync waves do not automatically sync child Applications.
No deletion finalizers are configured. Follow the explicit order in [bootstrap](bootstrap.md).

## Current limits and validation

Images and DNS names are placeholders; zero-replica workloads serve no traffic. Issuers, secret stores,
cloud identities, TLS, network isolation, persistent monitoring storage, and alert delivery are not
configured. A synced Gateway can provision a public load balancer; kind needs separate load-balancer
support or port forwarding. These are cluster implementation tasks, not shared guarantees.

CI parses YAML, builds every Kustomization, validates built-in Kubernetes schemas, checks rendered
Application paths, and renders charts with their effective values. Argo CD and Gateway API resources
skip schema validation because their schemas are not bundled. Runtime and cluster-specific checks must
be added alongside the features they verify. TechDocs builds separately in strict mode.
