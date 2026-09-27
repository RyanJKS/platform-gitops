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

Chart versions are pinned in the direct Applications under `catalog/platform/`.
Review release compatibility, render the selected charts, and verify readiness
before promoting a version. Shared values affect every cluster that selects them.
Root sync updates child Application definitions; each child's own sync policy
controls workload reconciliation. Do not assume parent sync waves order separate
child Applications. UK South's chart and issuer instead share one Application.
