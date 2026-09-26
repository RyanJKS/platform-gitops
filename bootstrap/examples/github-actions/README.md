# Optional ARC example

This example is not referenced by a bootstrap root. Do not apply it unchanged.

Copy `platform/` content into the selected cluster's `platform/github-actions/` directory and copy the
Application manifests into that cluster's `argocd/applications/`. Recalculate relative Kustomize paths,
replace example values paths with the new cluster paths, configure GitHub and image settings, and add
namespace/project permissions as described in the [onboarding guide](../../../docs/github-actions-runners.md).

The placeholder GitHub URL and image are intentional. `minRunners: 0` and `maxRunners: 0` drain the scale
set; a zero maximum does not mean unlimited capacity. Even a drained scale set starts a listener and
needs working GitHub credentials once synced. The examples only render in CI; nothing registers with GitHub.
