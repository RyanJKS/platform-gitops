# Onboarding an application

Application repositories build and publish images. This repository selects those images and configures
their deployment. Do not add application source code or image-build workflows here.

1. Add a reusable base under `catalog/applications/<application>/<component>/base/`. Include a
   Deployment, its Kustomization, and a Service only when the component receives traffic.
2. Add a cluster overlay under `applications/<application>/<component>/`. Reference the base using a
   relative path, set its namespace, and patch replicas and environment-specific settings.
3. Replace the `replace-me` image through the overlay's Kustomize `images` field. Prefer immutable
   digests. Set replicas above zero only after configuring runtime dependencies, secrets, and probes.
   Confirm the image supports non-root UID 10001 and the HTTP components listen on port 8080, or
   patch those defaults. The worker deliberately has no Service or HTTPRoute.
4. Define the namespace name once in `catalog/applications/<application>/namespace/`, then reference it
   from each target cluster's `argocd/kustomization.yaml`. Set namespace security labels
   in a cluster patch, include any required quota in the workload overlay, and select or create a shared workload AppProject with matching permissions.
   Grant only required namespaced resource kinds. ConfigMaps or secret resources need explicit
   AppProject permissions if added later. Keep secret values outside Git.
5. Add the component Application to `argocd/applications/` and list it in `argocd/kustomization.yaml`.
   Match the project, destination namespace, and source path. Use manual sync initially.
6. For HTTP traffic, add an HTTPRoute with your real hostname, backend Service, and Gateway parent.
   Supply Gateway API CRDs and a compatible Gateway/controller separately if using
   HTTPRoute; the former starter Gateway is no longer selected. Match route
   attachment permissions to the actual Gateway. For AKS application routing, use
   its existing ingress class and the certificate setup in [HTTPS](https.md).
7. Run `scripts/validate.sh`, merge, sync the root to register the Application, and sync its
   dependencies before the workload. Verify rollout, service endpoints, route status, and requests.

For example, add this field to a cluster API Kustomization, replacing the digest with the published value:

```yaml
images:
  - name: ghcr.io/ryanjks/atlas-market-api
    newName: ghcr.io/ryanjks/atlas-market-api
    digest: sha256:REPLACE_WITH_PUBLISHED_DIGEST
```

The configured Azure overlays inherit zero replicas from their catalog bases. Add a replica patch
only when enabling the workload. The default prevents unconfigured
placeholder images from being started but does not make the scaffold a working application.

## Sandbox webapp and MySQL

`catalog/applications/sandbox/webapp-mysql-pv/` is a sandbox with no bases or overlays.
Its root Kustomization includes `namespace.yaml`, `mysql/`, and `webapp/`. Each child has its own
Kustomization and sets `namespace: webapp-mysql`, so it can be applied independently.
The MySQL child includes its Deployment, Service, initialization and connection
ConfigMaps, PersistentVolumeClaim, and StorageClass. The webapp child includes its
Deployment and LoadBalancer Service. The MySQL Service is named `mysql`; both the
connection ConfigMap and webapp init container use this Service DNS hostname.
Rendering the root produces nine resources: the Namespace, six MySQL resources,
and two webapp resources. The child namespace setting applies to
the Deployment, Service, PVC, and initialization ConfigMap. The StorageClass is
cluster-scoped. The SQL database name `webappdb` is separate from the Kubernetes
namespace and does not need a namespace setting in SQL.

From the repository root, render the populated entry points without contacting a cluster:

```sh
kubectl kustomize catalog/applications/sandbox/webapp-mysql-pv
kubectl kustomize catalog/applications/sandbox/webapp-mysql-pv/mysql
kubectl kustomize catalog/applications/sandbox/webapp-mysql-pv/webapp
```

Before applying, select the intended sandbox cluster. Applying the root creates
the `webapp-mysql` namespace. Before applying a child independently, create the
namespace with `kubectl apply -f namespace.yaml` from the sandbox directory if it
does not already exist. The MySQL PVC requires a cluster with the
Azure Disk provisioner declared by `mysql/storage-class.yaml`. Confirm that the
cluster supports that provisioner and the configured storage parameters.

From the sandbox directory, choose the scope to apply:

```sh
cd catalog/applications/sandbox/webapp-mysql-pv
kubectl apply -k .         # Create the namespace and apply all populated children.
kubectl apply -k mysql/    # Apply only MySQL.
kubectl apply -k webapp/   # Apply only the webapp.
```

Applying a child does not delete resources from its sibling. A webapp that uses
MySQL still requires MySQL to be running. This webapp also references the
`mysql-configs` ConfigMap from the MySQL child, so apply MySQL before applying
the webapp independently. After applying MySQL, check
`kubectl -n webapp-mysql rollout status deployment/mysql-deployment` and
`kubectl -n webapp-mysql get pods,pvc,service`. If the PVC remains Pending,
inspect its events with `kubectl -n webapp-mysql describe pvc azure-managed-disk-pvc`
and confirm that the storage provisioner is available.
