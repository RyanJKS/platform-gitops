# Platform installation defaults

Import `argocd/` from a cluster's Argo CD Kustomization to select five common controller Applications:
Argo CD, cert-manager, External Secrets, monitoring, and Envoy Gateway.

This bundle contains installation defaults only. It does not create an AppProject, choose workload
namespace permissions, set quotas, or configure Gateway listeners. Each cluster supplies those choices.
Shared namespace bases contain names only; the shared gateway resource base contains only a GatewayClass.

Cluster overlays append Helm values for resource sizing, availability, retention, and other environment
settings. Put new identity, networking, TLS, storage, and access decisions under the relevant cluster.
Keep common versions and portable controller installation settings here. Add an opt-in policy profile
only when its consumers intentionally share the same policy and rollout lifecycle.

See [configuration boundaries](../../../docs/configuration-boundaries.md) before adding shared defaults.
