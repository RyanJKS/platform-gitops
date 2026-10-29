# AKS Envoy Helm bootstrap

The `helm-bootstrap.yaml` GitHub Actions workflow installs the Cilium and Argo CD
Helm charts for `clusters/azure/DEV-JKS/dev/uks/sandbox/aks-envoy`. It reads only
that cluster's bootstrap files. Cilium installs with Gateway API disabled;
Argo CD depends on the Cilium release becoming ready. Both releases use a
600-second Helm timeout. Once Argo CD is ready, its root selects the catalog
Gateway API Application to install the CRDs. Enable Cilium Gateway API through
a later reviewed Helmfile upgrade after those CRDs are established. The Argo CD chart seeds the
`all-apps` root Application through `extraObjects` in `bootstrap/values/argocd.yaml`.

This cluster keeps Cilium and Argo CD under Helmfile ownership, including later
upgrades. Gateway API CRDs are owned by Argo CD. The existing AKS AGIC
[Terraform handoff procedure](bootstrap.md)
continues to apply to AGIC, not to this cluster. Do not run the infrastructure
Argo CD Helm unit or create self-management Applications for the Envoy releases.
If Terraform already owns either release, review and remove that ownership
without uninstalling resources before using this workflow.
Do not run the infrastructure root Application unit for this cluster either.
If Terraform already owns `all-apps`, remove that state ownership without deleting
the Application before bootstrap adopts the root.

## App of apps root

The root uses the built-in `default` AppProject, repository
`https://github.com/RyanJKS/platform-gitops.git`, revision `dev/sandbox`, and path
`clusters/azure/DEV-JKS/dev/uks/sandbox/aks-envoy`. Its destination is this cluster's
`argocd` namespace. Change `extraObjects` if the repository or revision differs,
and publish the reviewed cluster selection to that revision before bootstrap.
Private repositories require repository credentials configured separately.

The chart renders its CRDs as ordinary templates. The root therefore uses Helm
`post-install,post-upgrade` hooks, which create it after CRDs and Argo CD are ready.
The `before-hook-creation` policy replaces the root on later Helmfile runs to
apply updated settings. It has no cascading-deletion finalizer, so replacing the
root leaves its children in place. Helm does not track hook objects as ordinary
release resources or automatically delete them on uninstall.

The root enables automatic sync and self-healing, with pruning disabled. Its
Kustomization selects AppProjects and child Applications; `bootstrap/` stays
excluded so Argo CD never adopts the Helm-managed controllers or the root itself.
It selects `catalog/platform/gateway-api` and patches that Application to use the
built-in `default` AppProject. Add reviewed workload selections to
`aks-envoy/kustomization.yaml` when ready. A restricted replacement AppProject must
allow the upstream Gateway API Git repository and cluster-scoped CRDs.

## Prepare Azure and GitHub

1. Provision AKS with BYOCNI: `network_plugin = "none"`. The current Envoy
   infrastructure configuration sets Azure CNI overlay; change and review that in
   the infrastructure repository first. This workflow does not migrate networking
   and rejects any cluster whose `networkProfile.networkPlugin` is not `none`.
2. Review the Cilium pod pool in `bootstrap/values/cilium.yaml`. The configured
   `10.244.0.0/16` pool is separate from the infrastructure VNet `10.2.0.0/16`
   and node subnet `10.2.1.0/24`. Also check the service CIDR, peered networks,
   and on-premises networks. This profile enables kube-proxy replacement;
   coordinate the AKS kube-proxy configuration in infrastructure. The workflow
   does not disable Azure-managed components or migrate an existing CNI.
3. Create GitHub environments `aks-envoy-plan` and `aks-envoy`. Keep the plan
   environment free of required reviewers so the diff can run first. Configure
   **Required reviewers** on `aks-envoy`, selecting your GitHub account or the
   appropriate deployment reviewers. Review the environment branch rules too.
   Register Azure federated identity credentials for both subjects:
   `repo:RyanJKS/platform-gitops:environment:aks-envoy-plan` and
   `repo:RyanJKS/platform-gitops:environment:aks-envoy`, with audience
   `api://AzureADTokenExchange`.
4. Give that identity permission to read AKS and obtain its user kubeconfig, such
   as Azure Kubernetes Service Cluster User Role. Grant Kubernetes cluster-admin
   access through your cluster's authorization model, such as Azure Kubernetes
   Service RBAC Cluster Admin for Azure RBAC-enabled clusters. Cilium and Argo CD
   install cluster-scoped resources. Prefer a separate plan identity with only
   the read access helm-diff needs, including Helm release Secrets, API discovery,
   and rendered resources. No admin kubeconfig or client secret is used.
5. Ensure the runner can reach the AKS API. The workflow uses `ubuntu-24.04`; a
   private API requires a runner with suitable network access and the same tools.

Set these variables in both GitHub environments, targeting the same cluster:

| Variable | Value |
| --- | --- |
| `AZURE_CLIENT_ID` | Client ID of the federated deployment identity |
| `AZURE_TENANT_ID` | Microsoft Entra tenant ID |
| `AZURE_SUBSCRIPTION_ID` | Subscription containing AKS |
| `AKS_RESOURCE_GROUP` | Resource group containing AKS |
| `AKS_CLUSTER_NAME` | Actual Azure AKS resource name, which can differ from `aks-envoy` |

The workflow installs Helm `3.18.6`, Helmfile `1.1.7`, helm-diff `3.12.5`, and kubelogin `0.2.12`.
It verifies the Helmfile archive checksum and uses the hosted runner's kubectl.
Authentication uses Azure CLI login through GitHub OIDC and kubelogin. The
kubeconfig is stored in the runner's temporary directory and removed afterward.

An environment name in YAML does not create an approval rule. Configure the
required reviewers in repository **Settings > Environments > aks-envoy**.
Both jobs check the environment policy through GitHub's API and stop if reviewers
are missing or the check fails. The workflow grants `actions: read` for that
check. See [GitHub deployment environments](https://docs.github.com/en/actions/how-tos/deploy/configure-and-manage-deployments/manage-environments)
for the protection settings and plan availability.

## Cilium Gateway API and Hubble

The cluster's `bootstrap/values/cilium.yaml` contains concrete network and
feature settings copied from the catalog: pod pool `10.244.0.0/16`, `/24` node
allocations, node initialization, VXLAN, kube-proxy replacement, ALPN, Hubble
relay/UI, metrics, and Grafana dashboards. Gateway API and GatewayClass creation start disabled
until the Argo CD-managed CRDs are established. Hubble selects AKS
system nodes using `kubernetes.azure.com/mode: system`, without assuming pool
names such as `systempool` or `sprotation`.

The direct API hostname is discovered from the selected AKS kubeconfig at runtime.
The workflow exports `CILIUM_K8S_SERVICE_HOST`; `values/cilium-api.yaml.gotmpl`
requires it and sets the API endpoint for both Cilium and upgrade preflight.
There is no placeholder or invented API hostname in cluster configuration.

The root selects `catalog/platform/gateway-api/application.yaml`, which reads
seven standard CRDs directly from upstream Gateway API `v1.6.3`. Argo CD uses
server-side apply and automatic pruning stays disabled. No Gateway API Helm
release remains. Review the Application's revision and schema changes alongside
Cilium upgrades. If Helm already owns these CRDs, review an ownership migration
before syncing; do not uninstall CRDs or leave both tools managing them. See the
[Cilium Gateway API guide](https://docs.cilium.io/en/stable/network/servicemesh/gateway-api/gateway-api/).
After enablement, the Cilium chart creates GatewayClass `cilium`; add Gateway
and HTTPRoute resources separately when routing workloads. This uses Cilium's Envoy integration. A
separate Envoy Gateway controller is not installed.

Hubble UI remains internal. Metrics endpoints and Grafana dashboard ConfigMaps
are enabled, but Prometheus and Grafana must be installed separately.
ServiceMonitors stay disabled because their CRDs are absent during bootstrap.
Configure Prometheus discovery before enabling ServiceMonitors later.

Reusable profiles, optional settings, and routing examples live under
`catalog/platform/cilium/` with `__PLACEHOLDER__` markers. Copy and adapt these
files; cluster bootstrap does not import the catalog profile automatically.

## Run and verify

Commit and push the reviewed workflow and cluster configuration. GitHub must
have the workflow on the default branch before it exposes manual dispatch.
Open **Actions > Helm bootstrap > Run workflow** and select the reviewed branch.
There are no automatic push or pull-request deployments. Runs for this cluster
are serialized without cancelling an installation already in progress.

The plan job runs `helmfile diff --detailed-exitcode --suppress-secrets` without
changing the cluster. Exit `0` means no changes; exit `2` means changes await
approval; any other exit code fails the job. Review the complete diff in the job
logs or download the `helm-bootstrap-diff` artifact, retained for three days.
Secret values are suppressed from the diff.

When changes exist, GitHub pauses the apply job at the `aks-envoy` environment.
Review the diff, then use **Review deployments** to approve or reject it. After
approval, `helmfile apply --suppress-secrets` diffs again and installs or upgrades
changed releases in dependency order. No changes skip the apply job. Both jobs
check out the exact workflow commit SHA, not a moving branch. Apply also verifies
that its AKS API hostname matches the target recorded by the plan job. Helmfile apply
re-evaluates current cluster state; the diff artifact is a review record, not a
saved executable plan. If cluster state changes materially while approval is
pending, cancel and rerun the workflow to review a fresh diff.

The Helmfile pins Cilium `1.20.3` in `kube-system` and Argo CD chart `10.9.0` in
`argocd`. The catalog Gateway API Application pins `v1.6.3`. Review chart and value
changes before every run. Cluster-specific
copies are independent of the examples under `catalog/platform/`.

To render locally with Helm 3 and Helmfile installed, run from the repository root:

```sh
CLUSTER=clusters/azure/DEV-JKS/dev/uks/sandbox/aks-envoy
export CILIUM_K8S_SERVICE_HOST=$(kubectl config view --minify -o jsonpath='{.clusters[0].cluster.server}' \
  | python3 -c 'import sys; from urllib.parse import urlsplit; print(urlsplit(sys.stdin.read()).hostname or "")')
helmfile --file "$CLUSTER/bootstrap/helmfile.yaml" template
kubectl kustomize "$CLUSTER"
```

The Kustomization renders the Gateway API Application in the built-in `default`
AppProject. It explicitly excludes the bootstrap directory; add future workload
Applications and AppProjects alongside that selection. The workflow seeds the root but does not configure private
repository access or install Envoy Gateway. Helm readiness does not wait for
the root Application or its children to finish reconciling.

For a local installation, use the same diff, review, and apply sequence:

```sh
helmfile --file "$CLUSTER/bootstrap/helmfile.yaml" diff --suppress-secrets
# Review the diff before confirming the interactive apply prompt.
helmfile --interactive --file "$CLUSTER/bootstrap/helmfile.yaml" apply --suppress-secrets
```

Helmfile does not install Gateway API CRDs. Argo CD installs them after bootstrap,
while Cilium Gateway API remains disabled. First-install diffs disable Kubernetes
API validation only for new Cilium and Argo CD releases; Helm's actual installation
still validates resources against the server. Rendering alone does not install
CRDs or validate connectivity.

The workflow checks Cilium, Envoy, Hubble, and Argo CD rollouts. After success,
select the intended AKS context locally and verify access:

```sh
helm -n kube-system list --filter '^cilium$'
helm -n argocd list --filter '^argocd$'
kubectl get nodes
kubectl -n argocd get application gateway-api
kubectl -n kube-system get deployments hubble-relay hubble-ui
kubectl -n argocd get pods
kubectl -n argocd get application all-apps
kubectl -n argocd port-forward service/argocd-server 8080:443
```

Open `https://localhost:8080`. Bootstrap uses Argo CD's internal TLS and ClusterIP
service, with ingress disabled. Obtain the initial admin password locally; do not
print it in workflow logs or commit it. Configure public routing separately.

## Enable Cilium Gateway API

After bootstrap, confirm the `gateway-api` Application is `Synced` and `Healthy`.
Resolve repository access and AppProject permission errors before proceeding. Wait for all seven CRDs:

```sh
kubectl -n argocd get application gateway-api
kubectl wait --for=condition=Established --timeout=120s \
  crd/gatewayclasses.gateway.networking.k8s.io \
  crd/gateways.gateway.networking.k8s.io \
  crd/httproutes.gateway.networking.k8s.io \
  crd/grpcroutes.gateway.networking.k8s.io \
  crd/referencegrants.gateway.networking.k8s.io \
  crd/backendtlspolicies.gateway.networking.k8s.io \
  crd/tlsroutes.gateway.networking.k8s.io
```

Set `gatewayAPI.enabled: true` and `gatewayAPI.gatewayClass.create: "true"` in
`bootstrap/values/cilium.yaml`. Review and commit that change, then use the same
Helmfile diff and approval workflow to upgrade Cilium. Verify the Cilium rollouts
and `kubectl get gatewayclass cilium` afterward. Add Gateway and HTTPRoute resources
separately. Keep CRD upgrades in the Gateway API Application, not Helmfile.

## Cilium upgrade preflight

The Helmfile includes `cilium-preflight`, disabled with `installed: false` for
fresh installs and normal runs. Cilium requires the
[preflight check before upgrades](https://docs.cilium.io/en/stable/operations/upgrade/#running-pre-flight-check-required)
to validate the existing installation and pre-pull the target images. The
preflight release shares only the direct API endpoint with the main release.
The agent and operator are disabled. It deliberately does not load the main
values file, avoiding duplicate node-init, Hubble, Gateway API, and dashboard
resources.

Run this sequence from the repository root with Helm 3, Helmfile, and access to
the intended AKS cluster. Do not run the bootstrap workflow during these steps.

1. Review the official upgrade guide and version-specific notes. Upgrade one
   minor release at a time, starting from the latest patch of the current minor.
2. Set only the `cilium-preflight` chart version to the target version and set
   its `installed` flag to `true`. Leave the main `cilium` release version unchanged.
3. Diff only preflight, review it, then approve the interactive apply and check
   both rollouts:

   ```sh
   BOOTSTRAP=clusters/azure/DEV-JKS/dev/uks/sandbox/aks-envoy/bootstrap/helmfile.yaml
   export CILIUM_K8S_SERVICE_HOST=$(kubectl config view --minify -o jsonpath='{.clusters[0].cluster.server}' \
     | python3 -c 'import sys; from urllib.parse import urlsplit; print(urlsplit(sys.stdin.read()).hostname or "")')
   helmfile --file "$BOOTSTRAP" --selector name=cilium-preflight diff --suppress-secrets
   helmfile --interactive --file "$BOOTSTRAP" --selector name=cilium-preflight apply --suppress-secrets
   kubectl -n kube-system rollout status daemonset/cilium-pre-flight-check --timeout=600s
   kubectl -n kube-system rollout status deployment/cilium-pre-flight-check --timeout=600s
   kubectl -n kube-system get daemonsets cilium cilium-pre-flight-check
   ```

   Confirm the preflight DaemonSet has as many ready pods as the running Cilium
   agents and the preflight Deployment is ready. Resolve any failures before
   proceeding. Enabling this release in a full bootstrap run does not establish
   a preflight gate before the main Cilium release.
4. Set `cilium-preflight.installed` back to `false`, review the selected diff,
   and confirm the selected interactive apply to remove preflight resources.
   An `installed: false` release is uninstalled,
   not merely skipped.
5. Update the main `cilium` chart version to the checked target version. If
   preserving settings from an older installation, configure `upgradeCompatibility`
   in `bootstrap/values/cilium.yaml` according to the upgrade guide. It is not
   needed for a fresh installation. Commit and push the reviewed changes, then
   run the bootstrap workflow and verify Cilium connectivity afterward.

## Recovery and changes

If Cilium fails, Helmfile stops before installing Argo CD. Inspect Helm status,
pod events, and controller logs using independent AKS access. Fix the cluster
values or networking prerequisite and rerun the workflow. A timeout can leave an
installed or failed Helm release, so inspect the release before retrying.

Review a known-good chart and values before recovery. Do not uninstall the CNI
or Argo CD to retry a failed bootstrap. A migration to another release owner
requires a separate ownership handoff; running two managers risks conflicting
updates. After installation, validate workload connectivity separately from Helm
readiness.
