# Adding a cluster or component

Provision the cluster and install Argo CD through its intended owner first. Azure
uses the Microsoft extension; local kind follows [its bootstrap guide](local-kind.md).
Never select the local Argo CD Application on an extension-managed cluster.

1. Create the cluster's `argocd/` entry point and `applications/` overlays.
2. Select only required direct Applications in `argocd/kustomization.yaml`.
   Reuse `catalog/` resources rather than copying shared manifests.
3. Include each workload's AppProject and namespace. Verify source repositories,
   destination namespaces, and cluster-resource permissions.
4. Configure an external root Application pointing at that exact `argocd/` path.
   Keep the root outside its own Kustomization and align Git revisions.
5. Run `scripts/validate.sh` and the strict TechDocs build. Review root pruning and
   existing resource ownership before applying a new selection.

UK South and local kind are current examples of direct platform selections.
Market demonstrates workload Applications and a shared namespace/AppProject.
AWS remains reserved; no root is selected there. The old ApplicationSet component
catalog has been removed, so do not add list-generator entries or references to it.

Do not select UK South's Azure HTTP-01 issuer on a cluster without its ingress
class and DNS prerequisites. Chart and issuer must remain in the same Application
so the readiness gate described in [HTTPS](https.md) can order their resources.

When retiring a component, removing its Git selection is not an uninstall plan.
Review live consumers, finalizers, and owner references first. Do not use an empty
Kustomization and root pruning to retire a deployed cluster.
