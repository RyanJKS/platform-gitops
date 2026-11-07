# Provider configuration templates

Provider examples are copy templates, not automatic values layers. Cloud-neutral
defaults live in [`../values.yaml`](../values.yaml).

Copy the required provider fields into a cluster Helm values file or an
Application's `spec.sources[0].helm.valuesObject`. Replace placeholders and review
resource ownership before deployment. Do not copy shared defaults or health checks.

See [Azure templates](azure/README.md) for AGIC ingress and Entra workload identity SSO.
