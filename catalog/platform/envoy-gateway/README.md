# Envoy Gateway

`application.yaml` installs Envoy Gateway through Argo CD using the official
Docker Hub OCI charts pinned to `v1.9.2`. It deploys the controller in
`envoy-gateway-system` and installs its `gateway.envoyproxy.io` extension CRDs.
The shared [Gateway API Application](../gateway-api/README.md) owns the
`gateway.networking.k8s.io` CRDs separately. The controller chart disables its
bundled CRDs, and the extension chart disables Gateway API CRDs, avoiding two
Applications managing the same definitions.

Envoy Gateway is a separate controller from Cilium's Envoy integration. Installing
this Application does not enable Cilium Gateway API or change Cilium networking.
It creates no GatewayClass instance, Gateway, routes, DNS records, or application
TLS certificates. The chart generates internal controller certificates.

## Provider configuration

`values.yaml` contains shared Kubernetes controller settings for Azure, AWS, GCP,
and local clusters. No provider-specific directory is required to install the
controller. The controller Service remains internal (`ClusterIP`). Its Service
settings do not configure the Envoy proxy Services created for Gateways.

Configure cloud load balancer annotations and proxy Service settings through
cluster-owned `EnvoyProxy` resources. Reference an `EnvoyProxy` from the
GatewayClass's `parametersRef` when needed. Keep cloud identities, network ranges,
DNS settings, TLS Secrets, and environment-specific scheduling in cluster files.
A cloud load balancer controller or a local LoadBalancer implementation must
provide the proxy's external address. See upstream
[EnvoyProxy configuration](https://gateway.envoyproxy.io/docs/api/extension_types/#envoyproxy).

## Select in Argo CD

Import the catalog directory into a cluster's Kustomization. Shared Gateway API
CRDs must already exist. Select `catalog/platform/gateway-api` only when Argo CD
owns those CRDs. AKS Envoy installs them through Helmfile and selects this
controller through its root; it must not select the Gateway API Application.
Patch the Application's project, destination, and
the values Git source revision (`spec.sources[2].targetRevision`) as needed. That
revision must contain these catalog files before syncing. Controller overrides
belong under `spec.sources[0].helm.valuesObject`; retain `crds.enabled: false`.
Keep both chart versions aligned when upgrading.

The default `platform-apps` AppProject must allow `docker.io/envoyproxy`, this
repository, and the `envoy-gateway-system` destination namespace. It must also
allow cluster-scoped CRDs, RBAC, and the chart's MutatingWebhookConfiguration.
Register the Helm repository as OCI-enabled in Argo CD if repository credentials
or explicit registration are required; the Application's Helm `repoURL` omits
the `oci://` prefix.

The Application uses wave `-3`, automatic sync and self-healing, server-side apply,
and disabled automatic pruning. The shared Gateway API Application uses wave
`-4`. Establish compatible Gateway API CRDs before syncing this controller. Parent
Application health checks are required to wait between child waves; existing
children still reconcile independently. Keep existing CRD ownership with one tool
and review any migration before syncing.

The shared Gateway API catalog currently selects seven standard CRDs. HTTP and
gRPC routing use that selection. If adding TCPRoute, UDPRoute, ListenerSet, or
experimental APIs, first extend the shared CRD selection and review compatibility
with the pinned controller. Do not enable the controller chart's bundled Gateway
API CRDs to add them.

Render the Application from the repository root:

```sh
kubectl kustomize catalog/platform/envoy-gateway
```

Create a cluster-owned GatewayClass from `samples/gatewayclass.yaml` after the
controller is ready. The sample uses `gateway.envoyproxy.io/gatewayclass-controller`
and is excluded from the catalog Kustomization. Add Gateways that select
`spec.gatewayClassName: envoy-gateway`, then HTTPRoutes with the corresponding
`parentRefs`. Configure listeners, hostnames, backend Services, and TLS in those
cluster-owned resources.

## Validation

With the intended cluster selected, verify the Application reports `Synced` and
`Healthy`, then check the controller:

```sh
kubectl -n argocd get application envoy-gateway
kubectl -n envoy-gateway-system rollout status deployment/envoy-gateway --timeout=600s
kubectl get crd envoyproxies.gateway.envoyproxy.io
```

After selecting the GatewayClass and routing resources, check GatewayClass
`Accepted`, Gateway `Accepted`/`Programmed`, and HTTPRoute
`Accepted`/`ResolvedRefs` conditions. Confirm the proxy Service has an address and
test DNS, HTTP, and HTTPS. Rendering does not prove controller readiness or cloud
load balancer access.

Review the upstream [installation guide](https://gateway.envoyproxy.io/docs/install/install-helm/)
and [release notes](https://gateway.envoyproxy.io/news/releases/notes/v1.9.2/)
before changing either chart pin or CRD ownership.
