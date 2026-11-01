# Shared Helm charts

Keep cloud-neutral charts directly under this directory and cloud-specific
charts under `<cloud>/`. Keep cluster source URLs, identities, vault settings,
and deployment choices in cluster values files.

- [`argocd`](argocd/README.md): root app-of-apps Application, separate from the official controller chart.
- [`azure/key-vault-secrets`](azure/key-vault-secrets/README.md): namespaces, ServiceAccounts, vault connections, and ExternalSecrets.

Each cluster references a chart from Helmfile and passes its own values file.
Preserve existing release names when moving chart sources. Reuse upstream
controller charts directly rather than copying their templates into this catalog.
See [AKS Envoy bootstrap](../../docs/helm-bootstrap.md#bootstrap-layout).
