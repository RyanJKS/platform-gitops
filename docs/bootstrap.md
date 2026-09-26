# Bootstrap

## Prerequisites

Start with an existing Kubernetes cluster, its explicit kubeconfig context, kubectl, Helm, and the
Argo CD CLI. Kubernetes 1.32 is the rendering baseline used by validation, not a tested compatibility
promise for every managed cluster version. Confirm each pinned chart supports your cluster version.
Allow outbound chart and Git access. Configure Argo CD repository credentials through a secure
channel if this repository is private. Never commit credentials.

Merge the configuration to `main` before bootstrap, or consistently change root and child revisions
to a review branch. All repository sources use `https://github.com/RyanJKS/platform-gitops.git`;
update every occurrence if using a fork.

Run commands from the repository root. All configured clusters currently use the shared Argo CD
values unchanged. If you add an Argo CD values override, append that same file to both its Application
and the bootstrap Helm command. The following example targets Azure dev's shared cluster:

```sh
CONTEXT=your-aks-shared-context
ROOT=azure-dev-uksouth-atlas-shared
kubectl --context "$CONTEXT" cluster-info
helm repo add argo https://argoproj.github.io/argo-helm
helm repo update argo
helm upgrade --install argocd argo/argo-cd --version 7.8.13 \
  --kube-context "$CONTEXT" --namespace argocd --create-namespace \
  -f catalog/platform/argocd/values.yaml --wait
kubectl --context "$CONTEXT" apply -f "bootstrap/roots/$ROOT.yaml"
kubectl --context "$CONTEXT" -n argocd port-forward service/argocd-server 8080:443 &
PORT_FORWARD_PID=$!
```

Keep this shell open. Log in with `argocd login localhost:8080` and the
bootstrap administrator credentials. The initial certificate is self-signed; inspect it before
accepting the CLI's certificate prompt. Retrieve the initial password through your normal secure
operator workflow. Rotate the bootstrap password and configure access controls before sharing access.

For the dedicated cluster, use its kubeconfig context and `azure-dev-uksouth-atlas-market` root.
For kind, create a cluster if needed with `kind create cluster --name kind-platform`, then use context
`kind-kind-platform` and root `local-kind`.

## Reconcile in order

If the cluster was bootstrapped before the directory rename, reapply its matching root manifest after
the new paths reach the tracked Git revision. This updates the root's source path. Sync the root to
update child Application paths before syncing children. Root and child Application names remain the
same; do not delete and recreate them for a directory move.

1. Sync the root with `argocd app sync "$ROOT"`. Verify AppProjects and child Applications appear.
   A root sync does not sync its children; sync waves are not a dependency scheduler here.
2. Sync `platform-namespaces`, then wait for its sync to complete.
3. Sync `platform-argocd`. This hands resource reconciliation to Argo CD using the same chart,
   release name, namespace, and values as bootstrap. Do not subsequently run Helm upgrade or
   uninstall against this release: stale Helm release metadata is not the source of truth.
4. Sync `platform-cert-manager`, `platform-external-secrets`, `platform-monitoring`, and
   `platform-gateway-controller`. Wait for controllers, webhooks, and CRDs to become ready.
5. Sync `platform-policy`, then `platform-gateway-resources`. Confirm the GatewayClass is accepted
   and the Gateway is programmed. A kind Gateway can remain pending until load-balancer support exists.
6. Follow [onboarding](onboarding-an-application.md) to configure workload images and replicas before
   syncing `atlas-ml-inference` or the dedicated cluster's `atlas-market-api` and `atlas-market-worker`.
   Local kind has no workload Applications by default.

For example, sync and wait for a controller with:

```sh
argocd app sync platform-cert-manager
argocd app wait platform-cert-manager --sync --health --timeout 300
```

The intended result is synced Applications with healthy controllers. Zero-replica workloads do not
serve traffic. HTTPRoutes use example DNS names; configure DNS and TLS separately before real use.
If a sync reports a missing custom resource kind, confirm the owning controller's CRDs are established,
then refresh and retry the dependent Application. See [recovery](recovery.md) for other failures.

After finishing, stop the background port-forward with `kill "$PORT_FORWARD_PID"`.
