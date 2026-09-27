# Configuration boundaries

The catalog supplies reusable definitions. A cluster explicitly selects components,
projects, and optional policy profiles, then supplies only its differences. A folder
name does not become a template variable or grant permission.

The Azure shared cluster variants (`uks` and `eus2`) select only `chaos-generator`,
using the existing `default` project. The `eus2` child reads the catalog base;
the `uks` child reads a cluster overlay that overrides replicas to two. Terraform
owns each cluster's `all-apps` root. The platform selections below apply to other
cluster examples. See [bootstrap](bootstrap.md#azure-shared-chaos-generator-only).

## Ownership and precedence

- `catalog/platform/components/` holds component inputs: existing names, chart
  repositories, pinned versions, releases, destination namespaces, and source paths.
- `catalog/platform/applicationsets/platform/` holds one reusable ApplicationSet
  template. It selects nothing until a cluster patches its list and cluster path.
- `catalog/platform/projects/` and `catalog/applications/*/project/` hold matching
  permission definitions. Cluster `argocd/projects/kustomization.yaml` adds only
  the appropriate workload destination to the platform project. Workload projects
  remain distinct; their permissions have not broadened.
- `catalog/platform/profiles/starter/` holds explicitly selected existing quota,
  HTTP Gateway, and namespace policy. These are starter choices, not production
  guarantees. Clusters can omit a profile or patch it without affecting others.
- `clusters/.../argocd/` selects components and projects. Unique workload Applications
  remain in `applications/`; shared platform Applications are generated.
- Cluster `platform/` and `applications/` hold resource selections and overrides.
  Initial roots remain at `bootstrap/root.yaml`, outside their own managed resources.

ApplicationSet list inputs take precedence through the supported `versionOverride`,
`namespaceOverride`, and `overrideValues` fields. Other component fields come from
one catalog input. `clusterPath` is supplied once in the cluster's ApplicationSet
patch; it prefixes local Kustomize paths and override value paths.

Helm loads the shared `defaultValues` file first, then each cluster-relative file
in `overrideValues`, in order. Later values win. Shared monitoring defaults retain
one Alertmanager replica, three-day retention, and Prometheus requests of 100m CPU
and 512Mi memory. Kind overrides only memory to 256Mi. Shared Gateway defaults
retain one replica. Chart versions remain unchanged; `versionOverride` permits a
staged cluster-specific upgrade.

Kustomize bases and components load before cluster patches. Existing clusters
explicitly select the starter profile: restricted Pod Security at `v1.30`, the
existing workload budget, and an HTTP listener accepting labeled namespaces.
The namespace component targets only `atlas-ml` and `atlas-market`. Review or
extend that target when introducing another workload namespace. AppProjects are
permission boundaries, not mirrors of subscriptions, environments, or regions.

## Cluster selection and installation ownership

The ApplicationSet matrix starts with the cluster's explicit component list. For
each entry, its Git-file generator reads exactly
`catalog/platform/components/{{ .component }}.yaml` from this public repository.
It never scans cluster directories. The cluster entry point patches the generator's
`values.clusterPath`; the Go template reads it as `.values.clusterPath`.

Azure clusters omit `argocd`. Terraform's Azure extension owns their Argo CD
installation, including ConfigMaps. Kind selects `argocd` and retains the existing
Helm-to-GitOps handover. No installation method changes in this refactor.

Review shared catalog changes as changes to every consumer. Shared policies are
opt-in and versioned with Git; matching values alone do not justify merging
separate security boundaries. Keep credentials, cloud identities, TLS choices,
network isolation, persistent storage, and recovery decisions under their owners.
