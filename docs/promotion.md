# Promotion

Promote configuration through reviewed pull requests. There is no automated environment promotion.
Azure production is a reserved scaffold; configure it with [adding a cluster](adding-a-cluster.md)
before attempting a production deployment.

1. Publish and verify an immutable application image in its source repository.
2. Update the dev overlay's image digest and relevant settings. Run validation, merge, sync, and verify
   application health, traffic, and dependencies in dev.
3. Open a separate pull request changing the destination environment to the same verified digest.
   Review environment-specific replicas, identities, DNS, secrets, and storage separately.
4. Merge and manually sync the destination Application during the appropriate change window.
   Record the Git revision, image digest, and observed result.

All configured Applications track `main`, including shared Argo CD defaults, Helm values, namespaces,
and workload bases. A catalog
change therefore affects every referencing cluster on its next sync. Keep environment-specific
rollouts in cluster overlays. For independent promotion of shared definitions, pin all Git sources
(including `$values`) to reviewed immutable revisions and explicitly advance them per environment.

Chart versions are pinned in the shared default Applications. To upgrade one dev cluster first,
patch its chart version in the cluster's Argo CD Kustomization, render its values, and review upstream
upgrade and CRD migration notes. After verification, update the shared default and remove the temporary
patch, or advance other cluster patches individually.
A Git revert does not necessarily reverse CRD migrations or restore data; see [recovery](recovery.md).
