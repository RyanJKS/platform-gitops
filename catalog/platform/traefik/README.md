# Traefik templates

`catalog/platform/traefik/` provides a reusable Application and alternative
values templates. Chart `42.1.2` runs Traefik `v3.7.16` and requires Kubernetes
`>=1.30`. The Application uses sync wave `-3`, automatic self-healing, and no
automatic pruning. No existing cluster selects this component.

Choose one routing mode:

- `configs/ingress/values.yaml` enables the Kubernetes Ingress provider and
  creates a named, non-default IngressClass. It disables Gateway API.
- `configs/gateway-api/values.yaml` enables Gateway API and creates a
  GatewayClass and Gateway. It disables the Kubernetes Ingress provider.

Both templates disable the Traefik custom resource provider and dashboard. The
Application sets `skipCrds: true`, so it does not install the chart's Traefik or
Hub CRDs. These profiles use standard Ingress or HTTPRoute resources instead of
Traefik IngressRoute and Middleware resources.

## Why CRDs and Helm tests are skipped

- `skipCrds: true` prevents Argo CD from installing CRDs bundled with the chart.
  Standard Kubernetes Ingress is a built-in API and needs no CRDs. Both profiles
  disable Traefik's custom resource provider, so they do not need the chart's
  Traefik CRDs. If you enable IngressRoute, Middleware, or other Traefik custom
  resources, install and manage those CRDs and enable `providers.kubernetesCRD`.
- Gateway API CRDs are a separate prerequisite. The pinned chart does not bundle
  them, so changing `skipCrds` to `false` would not install them. Install them
  before syncing the Gateway profile, as described below.
- `skipTests: true` omits Helm test resources. It does not skip controller
  installation or normal Kubernetes readiness and Argo CD health checks.
  Helm test resources are separate from those runtime checks.

## Create a cluster selection

Keep actual values under `clusters/`. For example, create these files under
`clusters/<cloud>/<account>/<environment>/<region>/<domain>/<cluster>/platform/traefik/`:

```text
traefik/
  kustomization.yaml
```

Create `kustomization.yaml` with a relative path from that directory to the
catalog base. Copy either catalog values template's complete mapping beneath
the patch's `value`, replacing `{}`, and fill its markers. For the cloud layout
above, the path and inline patch are:

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  - ../../../../../../../../../catalog/platform/traefik
patches:
  - target:
      group: argoproj.io
      version: v1alpha1
      kind: Application
      name: traefik
    patch: |-
      - op: replace
        path: /spec/sources/0/helm/valuesObject
        # Replace {} with the complete selected config mapping.
        value: {}
```

The catalog Application starts with `valuesObject: {}` so `op: replace` has an
existing field to replace. Fill the object before syncing; the empty object is
only a template. Indent the selected mapping under `value` in the inline patch.
The field name is `valuesObject`, not `valueObject`. No separate cluster
`values.yaml`, `valueFiles`, or values-only Git source is needed. The root
Application's Git revision selects the cluster Kustomization and its patch.

Add `traefik` to the parent platform Kustomization's `resources` when ready to
deploy. Before selecting it, update the cluster's `platform-apps` AppProject to
allow `https://traefik.github.io/charts` and the `traefik`
destination namespace. Existing AGIC sandbox permissions do not include Traefik.
Permit the chart's ServiceAccount, Service, Deployment, ClusterRole,
ClusterRoleBinding, and selected IngressClass or GatewayClass/Gateway resources.
Use rendered output to verify the complete resource allowlist. Cluster roots
must create the AppProject before this Application.

## Fill the selected values

For Ingress, set `ingressClass.name` and
`providers.kubernetesIngress.ingressClass` to the same class name. Set workload
Ingress `spec.ingressClassName` to that name. Keep hostnames, backends, and
`spec.tls` in cluster workload manifests. TLS Secrets belong in each Ingress's
namespace. The template does not create certificates.

For Gateway API, fill the GatewayClass name, Gateway name, listener hostnames,
workload namespace selectors, and TLS Secret name. The chart creates the Gateway
in `traefik`; create its TLS Secret in that namespace before syncing. Each
listener permits routes only from the selected workload namespace. Adjust the
selector deliberately if multiple namespaces need access.

Keep HTTPRoute resources in cluster workload paths. Set each route's
`parentRefs.name` to the chosen Gateway name and `parentRefs.namespace` to
`traefik`. Use `sectionName: web` or `websecure` to select the listener. Routes
must have hostnames compatible with the listener and reference real backend
Services. Keep TLS Secret contents outside Git; use your certificate or secrets
controller.

### Gateway API prerequisite

Install Gateway API CRDs before syncing the Gateway profile. The pinned chart's
[upstream instructions](https://github.com/traefik/traefik-helm-chart/tree/v42.1.2/traefik#gateway-api-crds)
include this exact command for the `v1.6.1` standard bundle:

```sh
kubectl apply -f https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.6.1/standard-install.yaml
```

This installs Gateway API definitions, not the Traefik controller. Apply it to
the intended cluster before syncing Traefik's Gateway profile, then wait for the
CRDs to become established. If another component already manages compatible
Gateway API CRDs, reuse that installation and keep its existing owner.

The chart is also listed on
[Artifact Hub](https://artifacthub.io/packages/helm/traefik/traefik); the pinned
upstream instructions above match this repository's chart version. Manage the
CRD prerequisite separately and confirm its compatibility with the target
cluster. The Traefik Application does not install it. Experimental routes remain
disabled. When the root also manages CRDs, ensure they are established before
Traefik's wave `-3` starts. Plain Ingress mode does not need Gateway API CRDs.

### Load balancer and DNS

Both profiles expose HTTP on Service port `80` and HTTPS on `443`, backed by
container ports `8000` and `8443`. Neither profile redirects HTTP to HTTPS.
Set cloud load balancer annotations and any Service settings in the cluster
copy. A cloud load balancer controller or another LoadBalancer implementation
must supply the Service address. Configure DNS separately. If using External
DNS with Gateway API, select its `gateway-httproute` source in that cluster.

## Local Setup

Run from the repository root with Helm, kubectl, and a kind cluster running
Kubernetes `>=1.30`. Select the intended kind context and confirm it before
installing:

```sh
kubectl config current-context
```

Install or upgrade Traefik using the local Ingress values:

If `providers.kubernetesCRD.enabled` is `true`, install the chart's CRDs using
the [custom resource setup](#traefik-custom-resources) below before this command.

```sh
helm upgrade --install traefik traefik \
  --repo https://traefik.github.io/charts \
  --version 42.1.2 \
  --namespace traefik \
  --create-namespace \
  --values clusters/local/kind/traefik-values.yaml \
  --skip-crds

kubectl -n traefik rollout status deployment/traefik --timeout=5m
```

The local values support standard Ingress and can also enable Traefik's custom
resource provider. Gateway API remains disabled, so Gateway API CRDs are not
required.
Helm test resources only run when explicitly invoked with `helm test`; the
install command does not need a `skipTests` flag.

### Traefik custom resources

Setting `providers.kubernetesCRD.enabled: true` enables Traefik's custom resource
provider; it does not install CRDs. The local Helm commands use `--skip-crds`,
which skips their installation on a new release. Removing that flag on an
existing release is not enough: Helm does not install missing chart CRDs during
an upgrade of an existing release.

To use IngressRoute, Middleware, or other Traefik custom resources, install the
CRD bundle from the same pinned chart version:

```sh
helm show crds traefik \
  --repo https://traefik.github.io/charts \
  --version 42.1.2 \
  | kubectl apply --server-side -f -

kubectl wait --for=condition=Established --timeout=60s \
  crd/ingressroutes.traefik.io
```

This bundle contains the chart's Traefik and Hub CRDs, not Gateway API CRDs.
Keep `providers.kubernetesCRD.enabled: true` in the local values and run the Helm
upgrade command above after the CRDs are established. Manage future CRD updates
separately from Helm release upgrades and keep them aligned with the chart.

### Application routing

The Service uses `LoadBalancer`. Without Cloud Provider KIND or MetalLB, its
external IP stays pending. For local access, keep this command running:

```sh
kubectl -n traefik port-forward svc/traefik 8080:80 8443:443
```

Use port `8080` for HTTP and `8443` for HTTPS. Preserve the hostname configured
in the Ingress, using a local hosts entry or an HTTP Host header. HTTPS also
requires the matching hostname and TLS certificate.

Workload Ingress resources must select `spec.ingressClassName: traefik` because
the local values create a non-default IngressClass. The local chaos-generator
overlay currently selects `nginx-ingress`; change the value in
`clusters/local/kind/chaos-generator/ingress-patch.yaml` to `traefik` before
applying that overlay for Traefik routing.

### Traefik dashboard

The local values disable the dashboard. Forwarding the Service's HTTP port
exposes application routing, not the dashboard; a `404 page not found` response
is expected when no Ingress matches the request.

To enable the dashboard for local development, run from the repository root:

```sh
helm upgrade --install traefik traefik \
  --repo https://traefik.github.io/charts \
  --version 42.1.2 \
  --namespace traefik \
  --create-namespace \
  --values clusters/local/kind/traefik-values.yaml \
  --skip-crds \
  --set api.dashboard=true \
  --set api.insecure=true

kubectl -n traefik rollout status deployment/traefik --timeout=5m
```

Forward the Deployment's internal dashboard port, keeping this command running:

```sh
kubectl -n traefik port-forward deployment/traefik 9000:8080
```

Open [the local dashboard](http://localhost:9000/dashboard/). The trailing slash
is required. This forwards pod port `8080`, not the Service's application port
`80`. No application Ingress is needed to view the dashboard.

`api.insecure=true` enables an unauthenticated API on the pod's internal port.
Use this only for local development. Keep `ports.traefik.expose.default: false`
so the Service does not expose that port, and keep port forwarding bound to
localhost. These CLI overrides do not change `traefik-values.yaml`. To disable
the dashboard and unauthenticated API, rerun the Helm command with
`--set api.dashboard=false --set api.insecure=false`.

## Validate before syncing

Run from the repository root, after filling the cluster files:

```sh
cluster_traefik=clusters/__PLACEHOLDER__/platform/traefik
kubectl kustomize "$cluster_traefik"
rg -n '__PLACEHOLDER__' "$cluster_traefik"
```

For chart validation, save the rendered Application's
`spec.sources[0].helm.valuesObject` mapping as `/tmp/traefik-values.yaml`, then run:

```sh
helm template traefik traefik --repo https://traefik.github.io/charts \
  --version 42.1.2 --namespace traefik --kube-version 1.30.0 \
  --values /tmp/traefik-values.yaml
```

Set `cluster_traefik` to the real directory and use the target Kubernetes version.
The marker search must return no matches. Also render the cluster root to check
selection and AppProject permissions. Rendering verifies composition and chart
values, not runtime readiness.

After syncing, verify the Application is `Synced` and `Healthy`, the Traefik
Deployment is available, and its Service has an address. For Gateway API, check
Gateway `Accepted`/`Programmed` and HTTPRoute `Accepted`/`ResolvedRefs` conditions.
Test DNS, HTTP routing, and HTTPS with the expected certificate from a client.
