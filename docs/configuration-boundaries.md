# Configuration boundaries

This repository is a starting platform configuration, not a complete production security policy.
Keep reusable installation mechanics in the catalog. Make deployment and policy decisions explicit
in each cluster. Missing production settings are future cluster work, not implicit shared guarantees.

## Where changes belong

| Layer | Owns | Does not own |
| --- | --- | --- |
| `bootstrap/roots/` | The mapping from one cluster to its Argo CD configuration | Infrastructure creation or workload policy |
| `catalog/platform/` | Common chart versions, controller installation values, namespace identities, reusable controller resources | Cluster access grants, listener exposure, TLS, resource budgets, identity bindings |
| `catalog/applications/` | Application namespace identities and portable workload contracts such as container names and service ports | Environment image releases, replicas, routing, credentials, cloud identities |
| `clusters/.../argocd/` | Capability selection, AppProjects, destinations, repository permissions, Applications and patches | Application source code or cluster provisioning |
| `clusters/.../platform/` | Namespace policy, quotas, controller sizing, gateways, TLS, storage, secret stores and cluster identity configuration | Secret values or cloud infrastructure state |
| `clusters/.../applications/` | Workload image versions, runtime overrides, replicas, routes and dependency configuration | Shared controller installation defaults |

Application base resource requests and non-root execution settings are starter workload defaults, not
enterprise policy. Override them when the actual application's runtime requirements are known. Platform
controller sizing and namespace budgets are cluster decisions and therefore already live under `clusters/`.

## Shared means portable, not mandatory

The shared Argo CD bundle installs five controller Applications. It does not supply a platform
AppProject or a Gateway listener. Each cluster supplies those resources explicitly. The shared
GatewayClass identifies Envoy's controller; the cluster Gateway defines exposure and route attachment.

Shared Namespace manifests contain names only. Cluster namespace patches select Pod Security Admission
labels and route eligibility. Quota amounts are defined directly in each cluster's policy directory.
Monitoring retention, resource requests, and controller replica counts are also cluster values.

As namespace configuration grows, group cluster-owned resources under
`platform/namespaces/<namespace>/` with its own `kustomization.yaml`. That group can reference the
catalog Namespace and include namespace-specific patches, Roles, and RoleBindings. Keep subjects and
access grants local unless they form an intentionally shared policy. ClusterRoles and ClusterRoleBindings
are cluster-scoped and belong with the relevant capability or cluster policy, not a namespace group.
Ensure only one Application owns each resource when splitting groups.

The existing dev and kind clusters deliberately repeat a few policy settings. These are independent
choices that happen to match today. Do not centralize them merely because their YAML is identical.
If several clusters later adopt a maintained policy with the same owner and rollout lifecycle, introduce
an explicit opt-in profile. Document its consumers and allow clusters to override or version it.

## Adding cluster-specific behavior

1. Put the resource or values file under the target cluster's `platform/` or `applications/` directory.
2. Include it in that cluster's Kustomization, or append the values file to the relevant Helm Application
   with an Argo CD Kustomize patch. Shared values load first and cluster values load last.
3. Update that cluster's AppProject permissions only if the new resource or destination requires them.
4. Add validation appropriate to the behavior. The current checks render manifests and validate built-in
   Kubernetes schemas; they do not verify cloud access, network reachability, TLS issuance, or recovery.
5. Update the relevant operational guide and record any prerequisites before enabling the capability.

For example, configure a TLS listener under `platform/gateway/resources/`, include the issuer/certificate
resources in an appropriate Application, and grant only the required project access. Installing
cert-manager alone does not configure certificates. Configure a secret store and workload identity
before adding ExternalSecret consumers; never put secret values in Git.

## Work to complete before a cluster serves production traffic

The cluster owner decides and implements:

- Operator SSO/RBAC, AppProject permissions, repository access, and workload identity bindings.
- Network isolation, Gateway exposure, DNS, TLS, and permitted route namespaces.
- Resource budgets, controller availability, storage classes, persistence, and backup/restore procedures.
- Monitoring retention, alert routing, service objectives, and incident access.
- Image releases, runtime settings, promotion rules, and cluster-specific integration or policy checks.

The supplied platform projects remain broad starter administrator permissions. Their location makes
that choice visible per cluster; moving the files does not harden them. The supplied HTTP Gateways and
namespace policies likewise preserve the existing starter behavior. AWS and Azure production remain
unconfigured scaffolds until their owners implement the required settings.
