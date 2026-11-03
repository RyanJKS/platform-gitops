# Gateway API CRDs

`application.yaml` lets Argo CD manage seven Gateway API standard CRDs for the
Cilium and Envoy Gateway profiles. It reads upstream YAML directly from
[`kubernetes-sigs/gateway-api` at `v1.6.3`](https://github.com/kubernetes-sigs/gateway-api/tree/v1.6.3/config/crd/standard).
The source filter selects GatewayClass, Gateway, HTTPRoute, GRPCRoute,
ReferenceGrant, BackendTLSPolicy, and TLSRoute definitions. It excludes other
upstream resources. No local Helm chart or vendored CRDs are needed.

This Application installs no gateway controller, GatewayClass instance, Gateway,
routes, load balancer, DNS records, or certificates. Select a compatible controller
such as Cilium or Traefik separately.

## Provider configuration

The CRDs are identical on Azure, AWS, GCP, and local clusters, so no
`configs/<provider>/` directories are needed. `values.yaml` records this explicitly;
it is informational and is not consumed by this raw-manifest Application.
Configure cloud authentication, load balancer annotations, networking, and routing
in the selected controller's cluster configuration.

## Select in Argo CD

Import this directory from the cluster's Kustomization. The Application defaults
to the `platform-apps` AppProject, namespace `argocd`, and the local Kubernetes
cluster. Patch these settings for the deployment. The AppProject must allow
`https://github.com/kubernetes-sigs/gateway-api.git` and cluster-scoped
`apiextensions.k8s.io/CustomResourceDefinition` resources.

The Application uses wave `-4`, server-side apply for large CRD schemas, automatic
sync, and self-healing, with pruning disabled. It has no cascading-deletion
finalizer. Keep one owner for these CRDs; if Helm or another tool already manages
them, review an ownership migration before syncing. Disabling automatic pruning
does not protect against manual pruning or direct CRD deletion.

Render the Application from the repository root:

```sh
kubectl kustomize catalog/platform/gateway-api
```

Sync it and wait for `Synced` and `Healthy`, then verify the selected CRDs:

```sh
kubectl wait --for=condition=Established --timeout=120s \
  crd/gatewayclasses.gateway.networking.k8s.io \
  crd/gateways.gateway.networking.k8s.io \
  crd/httproutes.gateway.networking.k8s.io \
  crd/grpcroutes.gateway.networking.k8s.io \
  crd/referencegrants.gateway.networking.k8s.io \
  crd/backendtlspolicies.gateway.networking.k8s.io \
  crd/tlsroutes.gateway.networking.k8s.io
```

Install or enable the controller's Gateway API support only after these CRDs are
established. A parent needs Application health checks to wait between child waves;
existing children still reconcile independently.

AKS Envoy selects this Application through its app-of-apps root at wave `-4`,
before Envoy Gateway at `-3`. Cilium's Gateway API support remains disabled, so
Cilium and Argo CD can start without these CRDs. The root's Application health
check waits for the CRD child before advancing. Envoy Gateway's child installs
only its own extension CRDs. This avoids storing shared CRD templates in a Helm
release Secret and needs no chart packaging helper. See the
[bootstrap guide](../../../docs/helm-bootstrap.md).

## Updating the CRDs

Review the controller's compatibility requirements, upstream upgrade notes, and
schema changes before changing `spec.source.targetRevision`. Review the directory
filter at the new tag as well. When this Application is selected, Argo CD owns CRD upgrades. AKS Envoy uses
that model while Helmfile owns its Cilium and Argo CD releases. Never delete a
CRD to resolve an upgrade error: deletion also
deletes its custom resources.
