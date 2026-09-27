# Recovery

## Failed sync

Inspect `argocd app get <name>` and Kubernetes events in the destination namespace. Confirm the
Application's repository revision, path, project permissions, and chart version. For missing CRDs or
webhooks, restore the owning controller before retrying dependent resources. For `ImagePullBackOff`,
check the digest and registry credentials; never enable placeholder images.

Fix configuration in Git, merge, refresh the affected Application, and sync again. Manual cluster edits
are emergency measures and must be reconciled back to Git before a later sync overwrites them.

## Roll back configuration

Revert the relevant change in a new pull request, validate it, merge, and manually sync the affected
Application. Review shared catalog consumers. Rollbacks of image versions or manifest configuration
do not restore databases, persistent volumes, or undo incompatible CRD migrations.

Pruning is not enabled automatically. A file removed from Git remains deployed until you explicitly
review and prune it. Inspect the diff before `argocd app sync <name> --prune`, especially for namespaces,
CRDs, and persistent storage. Deleting a namespace also deletes its namespaced resources.

## Recover Argo CD or a cluster

Restore a healthy cluster and its credentials first. Install the same Argo CD chart and values using
[bootstrap](bootstrap.md), then apply only that cluster's root. Restore repository credentials and
operator access from a secure backup. Reconcile platform capabilities and workloads in bootstrap order.

Git contains desired manifests, not cloud infrastructure, secret values, or application data.
Back up those systems separately and test restoration. Monitoring storage is ephemeral by default.
Do not assume recreating the cluster restores historical metrics or workload data.

## Remove an Application

Roots and children omit resource deletion finalizers. Deleting a root does not cascade into child
Applications, and deleting a child does not automatically remove its workloads. Inventory live resources,
remove the Git references, then explicitly plan resource deletion or retention. Do not add a cascading
finalizer as a shortcut for cluster retirement.
