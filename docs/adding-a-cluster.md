# Adding a cluster

Read [configuration boundaries](configuration-boundaries.md) first. Shared defaults install controllers;
the cluster owns deployment and policy decisions.

1. Provision the cluster, node pools, networking, and identity outside this repository.
2. Create `clusters/<cloud>/<subscription-or-account-alias>/<environment>/<region>/<spoke>/<cluster>/` with `argocd/`,
   `platform/`, and `applications/`. Use the shorter existing convention for local clusters.
   Use a stable subscription alias for Azure or account alias for AWS. Map it to the actual cloud ID
   in infrastructure configuration. Place environments beneath that subscription or account alias. Use a separate alias when an
   environment runs in a different subscription or account.
3. Copy the closest cluster overlay and review its policy choices. Update Application paths, workload
   namespaces and hostnames, and relative Kustomize references. Do not assume another cluster's
   permissions, security labels, quotas, sizing, or HTTP listener are appropriate for this cluster.
4. Import `catalog/platform/defaults/argocd/` in the Argo CD Kustomization. Add local platform and
   workload AppProjects with explicit destinations and permissions, plus the namespace, policy,
   gateway-resource, and workload Applications. Omit unneeded controllers with deletion patches.
5. Compose shared namespace identities and cluster-owned policy patches under `platform/namespaces/`.
   Set resource budgets directly under `platform/policy/`. Add only the workload namespaces required.
6. Append local Helm values for controller sizing, retention, availability, or provider settings.
   The configured clusters demonstrate the monitoring and gateway patches. Keep shared chart versions
   unless a staged upgrade requires a cluster-specific version patch.
7. Import the shared GatewayClass into `platform/gateway/resources/` and define this cluster's Gateway.
   Implement exposure, TLS, route permissions, identities, secret stores, storage, and network policies
   as required. Include resources in Applications and add corresponding validation and documentation.
8. Add a uniquely named root in `bootstrap/roots/` pointing to this cluster's `argocd/` path.
9. Run `scripts/validate.sh`, review rendered resources, update the bootstrap table and architecture
   inventory, then follow [bootstrap](bootstrap.md) using the matching context.

AWS dev and Azure prod are placeholders. Replace their README files with configuration only when
implementing those clusters. Do not use empty Kustomizations to retire existing clusters: pruning them
could delete resources.
