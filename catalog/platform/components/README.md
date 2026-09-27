# Platform component inputs

Each YAML file supplies one component's existing Application name, sync wave,
destination namespace, and source defaults. Helm inputs pin chart repository,
version, release, and the shared values file. Kustomize inputs supply a path
relative to the consuming cluster.

The shared template is in `../applicationsets/platform/`. It selects nothing by
default. A cluster explicitly lists component names and supplies one cluster path.
Supported element overrides are `versionOverride`, `namespaceOverride`, and ordered
`overrideValues` paths relative to that cluster. Do not use wildcard component names.

See [configuration boundaries](../../../docs/configuration-boundaries.md) for
precedence and [bootstrap](../../../docs/bootstrap.md) for migration requirements.
