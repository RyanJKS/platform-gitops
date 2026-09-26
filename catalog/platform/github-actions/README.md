# GitHub Actions runners

Optional Actions Runner Controller (ARC) capabilities. No configured cluster or default bundle enables them.

- `controller/`: portable controller logging defaults and the `arc-systems` namespace base.
- `runner-scale-set/`: the required runner container command and the `arc-runners` namespace base.

The controller and scale set are separate upstream Helm charts. Their pinned `0.13.0` Applications and
cluster-specific sample values are under `bootstrap/examples/github-actions/`. Treat that version as a
reviewable starting point, not a claim that it is the latest release.

GitHub URLs, secret references, runner groups, capacity, image versions, controller service accounts,
and network/storage policy belong in cluster configuration. No credential values belong in this catalog.
See [runner onboarding](../../../docs/github-actions-runners.md) for setup and validation.
