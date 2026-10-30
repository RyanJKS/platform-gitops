# Cilium

This directory contains reusable Helm values and installation guidance. It has no
Argo CD `application.yaml` or root Kustomization. Copy the relevant sample into a
cluster bootstrap directory, adapt it, and select the chart in that cluster's
Helmfile. Nothing installs directly from this catalog directory.

## Reusable examples

- `configs/azure/values.yaml`: AKS BYOCNI profile with Cilium Gateway API, Hubble,
  metrics, dashboards, and commented options for UI ingress and ServiceMonitors.
- `catalog/platform/gateway-api/`: Argo CD Application for pinned upstream
  Gateway API CRDs. Establish these CRDs before enabling Cilium Gateway API.
- `samples/gateway.yaml` and `samples/httproute.yaml`: HTTP routing examples with
  a commented HTTPS listener. These samples are not installed by the Gateway API Application.

Replace every `__PLACEHOLDER__` in copied files according to its field. Keep
cluster identities, network ranges, hostnames, and node pool names in the cluster
copy. Catalog examples are not referenced automatically by cluster bootstrap.

## AKS BYOCNI profile

`configs/azure/values.yaml` targets an AKS cluster created with
`network_plugin = "none"`. It uses Cilium cluster-pool IP address management,
VXLAN tunneling, Cilium kube-proxy replacement, node initialization, and one
operator replica for a sandbox. Supply a non-overlapping pod pool and the direct
Kubernetes API hostname/IP. The API hostname must not include `https://` or use
the Kubernetes Service IP. Port `443` is the AKS API port. Node IPs still come
from the VNet subnet; cluster-pool pod IPs use a separate range.

A `/16` pool split into `/24` allocations provides 256 node blocks. This is not
a guarantee of 256 usable pod addresses per node or 256 nodes supported by AKS.
Check the service CIDR, connected networks, AKS limits, and node capacity.
Review operator availability for production clusters.

This is not a configuration for Azure CNI overlay or Azure-managed Cilium.
Do not install another CNI over either. Provision BYOCNI through your
infrastructure repository before using this sample. See the official
[Cilium Helm installation guide](https://docs.cilium.io/en/stable/installation/k8s-install-helm/).

## Gateway API and observability

Establish compatible Gateway API CRDs before enabling Cilium Gateway API.
The [Gateway API Application](../gateway-api/README.md) pins `v1.6.3` for the
Cilium `1.20.3` cluster example; review both versions together when upgrading.
On a fresh cluster where Argo CD needs Cilium networking, bootstrap Cilium with
`gatewayAPI.enabled: false` and `gatewayAPI.gatewayClass.create: "false"` first.
After Argo CD establishes the CRDs, enable both settings through Helmfile.
The catalog profile enables them for clusters that already meet this prerequisite. See the official
[Cilium Gateway API guide](https://docs.cilium.io/en/stable/network/servicemesh/gateway-api/gateway-api/).
The chart creates the `cilium` GatewayClass. It does not create Gateways,
HTTPRoutes, DNS records, or TLS certificates. Copy and adapt the routing samples
when needed. This uses Cilium's Envoy integration, not a separate Envoy Gateway
controller installation.

The profile enables Hubble relay/UI and DNS, drop, TCP, flow, ICMP, and HTTP
metrics. `enableAlpn: true` enables HTTP/2 TLS negotiation and Service
`appProtocol` support. Set backend Service `appProtocol` appropriately for HTTP/2.
`socketLB.hostNamespaceOnly` limits socket load balancing to the host namespace.

Replace the catalog's pool markers with existing AKS pool names, or select system
nodes with `kubernetes.azure.com/mode: system`, as the Envoy cluster does.
`CriticalAddonsOnly` tolerations allow Hubble onto tainted system nodes.
Hubble UI stays a ClusterIP Service unless you explicitly configure access.

Metrics and Grafana dashboard ConfigMaps are enabled. This does not install
Prometheus or Grafana. ServiceMonitors stay disabled until Prometheus CRDs and
matching discovery labels exist. The chart's annotated metrics endpoints can be
scraped by a suitably configured Prometheus. Enable chart ServiceMonitors later
only with a deliberate discovery configuration to avoid duplicate scraping.

For example, from the repository root:

```sh
CLUSTER=clusters/azure/DEV-JKS/dev/uks/sandbox/aks-envoy
mkdir -p "$CLUSTER/bootstrap/cilium"
cp catalog/platform/cilium/configs/azure/values.yaml "$CLUSTER/bootstrap/cilium/values.yaml"
# Replace markers in the copied files and review chart/CRD versions.
# The Envoy Helmfile also requires the real API hostname in this variable.
export CILIUM_K8S_SERVICE_HOST=$(kubectl config view --minify -o jsonpath='{.clusters[0].cluster.server}' \
  | python3 -c 'import sys; from urllib.parse import urlsplit; print(urlsplit(sys.stdin.read()).hostname or "")')
helmfile --file "$CLUSTER/bootstrap/helmfile.yaml" template
helmfile --file "$CLUSTER/bootstrap/helmfile.yaml" diff --suppress-secrets
# Review the diff, then confirm the prompt to apply.
helmfile --interactive --file "$CLUSTER/bootstrap/helmfile.yaml" apply --suppress-secrets
```

The AKS Envoy Helmfile installs Cilium `1.20.3` in `kube-system` with Gateway API
disabled and waits before installing Argo CD. The root then selects the
[Gateway API Application](../gateway-api/README.md). Follow the
[bootstrap procedure](../../../docs/helm-bootstrap.md#enable-cilium-gateway-api)
to enable Gateway API after the CRDs are established. Keep version upgrades in the
cluster Helmfile;
this catalog does not select a version or receive automatic cluster overrides.
The catalog sample and cluster copy evolve independently.

## Verification

After bootstrap succeeds, use the intended cluster context:

```sh
helm -n kube-system list --filter '^cilium$'
kubectl -n kube-system rollout status daemonset/cilium --timeout=600s
kubectl -n kube-system rollout status deployment/cilium-operator --timeout=600s
kubectl -n kube-system rollout status deployment/hubble-relay --timeout=600s
kubectl -n kube-system rollout status deployment/hubble-ui --timeout=600s
kubectl get nodes
```

After enabling Gateway API, also verify `kubectl get gatewayclass cilium`.
Review pod events and Cilium logs if nodes or pods remain unready. A Helm render
does not prove packet connectivity; use the Cilium connectivity checks for your
cluster before deploying workloads. See the
[bootstrap operations guide](../../../docs/helm-bootstrap.md) for ownership and
recovery.
