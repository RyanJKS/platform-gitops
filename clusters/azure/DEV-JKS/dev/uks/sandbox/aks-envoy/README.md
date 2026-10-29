# AKS Envoy sandbox

## Bootstrap

The [Helm bootstrap workflow](../../../../../../../.github/workflows/helm-bootstrap.yaml)
uses `bootstrap/helmfile.yaml` to install Cilium, then Argo CD. It runs Helmfile
diff first, waits for approval in the `aks-envoy` GitHub environment, then runs
Helmfile apply. Both releases remain Helm-owned. Cilium starts with Gateway API
disabled so networking can become ready before Argo CD installs Gateway API CRDs.
Argo CD chart values also seed the `all-apps` root Application, pointing at this
folder on `dev/sandbox`.

Helmfile deployment: https://helmfile.readthedocs.io/en/latest/

Edit chart versions in the Helmfile and cluster configuration in
`bootstrap/values/`. Cilium values are a cluster-owned copy of the
[catalog example](../../../../../../../catalog/platform/cilium/configs/azure/values.yaml).
Cilium profile edits do not change this cluster automatically. The root selects
the shared Gateway API Application, so catalog Application changes do affect it.
After that Application is Synced and Healthy and its CRDs are established, enable
`gatewayAPI.enabled` and `gatewayAPI.gatewayClass.create` in the cluster Cilium
values through a reviewed Helmfile upgrade. Follow the
[enablement procedure](../../../../../../../docs/helm-bootstrap.md#enable-cilium-gateway-api).
The pod pool is `10.244.0.0/16`; Hubble uses AKS system nodes. The direct API
hostname is discovered from kubeconfig and supplied through a required runtime
value, rather than stored as a placeholder. See the
[Cilium Gateway API guide](https://docs.cilium.io/en/stable/network/servicemesh/gateway-api/gateway-api/)
for controller prerequisites.

AKS must use BYOCNI (`network_plugin = "none"`). The current infrastructure
configuration uses Azure CNI overlay and needs a separately reviewed change before
this bootstrap can run. The workflow rejects other network plugins before
installing charts. Review the pod CIDR against your Azure network ranges.

Configure Azure OIDC, the `aks-envoy-plan` environment, and required reviewers on
the `aks-envoy` environment as described in
[AKS Envoy Helm bootstrap](../../../../../../../docs/helm-bootstrap.md).
The AKS resource name comes from that environment; it can differ from this folder.

## GitOps boundary

The cluster `kustomization.yaml` selects the catalog Gateway API Application and
patches its project to the built-in `default` AppProject. Add reviewed AppProjects
and workload Applications there when needed. It does not include `bootstrap/`,
Argo CD self-management, or a Cilium Application. The `all-apps` root reconciles
only those selected resources. Do not run the infrastructure
Argo CD Helm or root Application units against this cluster while Helmfile
owns bootstrap.

The root is created through the chart's `extraObjects` after Argo CD is ready.
Configure repository credentials separately if the repository is private. Envoy
Gateway is not installed by bootstrap. Workload notes remain in `workloads/`.
