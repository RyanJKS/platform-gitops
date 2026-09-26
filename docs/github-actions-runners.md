# GitHub Actions runners

Actions Runner Controller (ARC) is an optional platform capability. The catalog holds portable defaults
and namespace definitions; cluster configuration owns GitHub registration, capacity, runtime, and policy.
No current cluster enables ARC, and no default bundle includes it.

## Layout and ownership

```text
catalog/platform/github-actions/
  controller/
    values.yaml
    namespaces/
  runner-scale-set/
    values.yaml
    namespaces/

clusters/<target>/platform/github-actions/
  controller/values.yaml
  runner-scale-sets/<name>/values.yaml
```

A reviewed example is provided at `bootstrap/examples/github-actions/`. Its two Applications pin
`gha-runner-scale-set-controller` and `gha-runner-scale-set` to `0.13.0` from
`ghcr.io/actions/actions-runner-controller-charts`. Upgrade both deliberately after checking upstream
compatibility. Runner images are versioned independently and must be supplied by the cluster owner.

The catalog namespaces have no security labels or grants. Apply those choices in the cluster. One
Application must own each Namespace; normally compose both namespace bases through the existing
`platform-namespaces` Application. Separate runner scale sets can use dedicated namespaces instead of
sharing `arc-runners` when their trust or permission boundaries differ.

## Enable on one cluster

1. Copy the example controller and scale-set values into the selected cluster's
   `platform/github-actions/` directory. Copy its two Application manifests into the cluster's
   `argocd/applications/`, and list them in the cluster's Argo CD Kustomization.
2. Change each Application's second `$values` path from the example directory to the copied cluster
   values file. Keep the shared catalog file first. Copying the examples alone does not enable them.
3. Add the controller and runner namespace bases to the cluster's namespace Kustomization. Calculate
   relative paths from that directory; the example namespace Kustomization demonstrates composition,
   but its relative paths are specific to the example location.
4. Add the OCI repository to the cluster platform project's `sourceRepos`, and allow destinations
   `arc-systems` and `arc-runners`. Review ARC's rendered CRDs and RBAC if narrowing the starter platform
   project permissions. Define node selection, network policy, resource budgets, and namespace policy
   for the intended workflow trust boundary before running jobs.
5. Set `githubConfigUrl` to the intended GitHub organization or repository URL. Configure an appropriate
   runner group for organization runners when needed. Set `runnerScaleSetName` to the label workflows
   will use, such as `platform-builds`. Give each scale set a distinct Application and Helm release name.
6. Create the referenced `github-actions-app` Secret in the runner namespace using an external secret
   system or secure operator process. For GitHub App authentication, provide `github_app_id`,
   `github_app_installation_id`, and `github_app_private_key` with the required GitHub permissions.
   The example contains a Secret name only, never credentials. Enterprise-level registration has
   different authentication support; check the upstream documentation before selecting it.
7. Replace the runner image placeholder with an approved version or digest. The runner container must
   remain named `runner`. The example explicitly supplies `controllerServiceAccount.namespace` and
   `.name`, matching the controller chart's configured service account. This avoids Helm lookup-based
   controller discovery, which is unavailable during Argo CD rendering.
8. Choose capacity. The example sets both minimum and maximum to zero, which drains the scale set.
   Set a finite positive maximum only when ready to execute workflows. A listener still needs working
   credentials even when runner capacity is zero.
9. Run `scripts/validate.sh`, merge, sync the root, and sync namespaces and credentials first. Sync the
   controller and wait for its Deployment and ARC CRDs to be ready. Then sync the scale-set Application.
   Sync-wave annotations do not replace this order because child Applications use manual sync.

For example, after connecting the Argo CD CLI to the selected cluster:

```sh
argocd app sync github-actions-controller
argocd app wait github-actions-controller --sync --health --timeout 300
argocd app sync github-actions-platform-builds
```

Check listener and runner logs, confirm the scale set is available in GitHub, and run a trusted test
workflow using `runs-on: platform-builds`. No GitHub registration or workflow execution is performed by
repository validation.

## Execution mode and future checks

The starter uses the chart's ordinary runner mode. It does not configure Docker-in-Docker, a Docker
socket, privileged containers, Kubernetes job-container hooks, or storage. Docker actions and container
jobs require a deliberate execution-mode configuration. Add its required isolation, permissions,
storage, and validation in the target cluster; do not impose them on every catalog consumer.

Treat workflow execution as a separate trust boundary from application workloads and platform
credentials. Decide which repositories and pull requests may use each runner group. Keep jobs needing
different access in separate scale sets and, where needed, separate nodes or clusters.

CI renders both optional charts with the example values and builds their namespace compositions. This
checks chart inputs and paths, not GitHub permissions, runtime scheduling, or job isolation. Add a trusted
integration workflow and operational checks when enabling a cluster. To pause allocation, set both
`minRunners` and `maxRunners` to zero and follow upstream guidance for draining active jobs before removal.

See [GitHub's ARC documentation](https://docs.github.com/en/actions/tutorials/use-actions-runner-controller)
for authentication permissions, execution modes, and version-specific operations.
