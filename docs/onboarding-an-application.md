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

The UK South sandbox Argo CD Application `webapp-mysql-pv` watches the
`dev/sandbox` revision and renders
`clusters/azure/DEV-JKS/dev/uks/sandbox/aks-app-routing/applications/webapp-mysql-pv/`.
This cluster overlay sets `webapp-deployment` to three replicas and leaves MySQL
at one replica. It also selects `StandardSSD_LRS` for the Azure Disk StorageClass
because `Standard_D2_v5` nodes cannot attach `Premium_LRS` disks. The legacy
StorageClass name `managed-premium-retain-sc` stays unchanged to preserve existing
PVC references; the catalog default remains `Premium_LRS`.
The webapp container requests `250m` CPU and `512Mi` memory, with limits of one CPU
and `1Gi` memory. `JAVA_OPTS=-Xms128m -Xmx512m` caps the Java heap at `512Mi`,
leaving memory for native allocations, thread stacks, and Tomcat. These settings
are specific to the UK South overlay; the catalog resource defaults stay unchanged.
Kustomize replica targets must match the Deployment's
`metadata.name`, not its container name or Pod labels.

Render the same source as Argo CD before committing changes:

```sh
kubectl kustomize clusters/azure/DEV-JKS/dev/uks/sandbox/aks-app-routing/applications/webapp-mysql-pv
```

If Argo CD reports `Unknown` with a `ComparisonError`, inspect
`kubectl -n argocd get application webapp-mysql-pv -o yaml` in the intended
cluster context. A replica target named `webapp` fails manifest generation
because the Deployment is named `webapp-deployment`. Correct the target in Git,
commit and push to `dev/sandbox`, then refresh the Application with
`argocd app get webapp-mysql-pv --hard-refresh` using the matching Argo CD server.
The Application uses automated sync. Disabled sync retries do not cause this
manifest-generation failure.

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

### Diagnose webapp restarts and browser access

Use these commands in the intended sandbox cluster context:

```sh
kubectl -n webapp-mysql get pods -l app=webapp
kubectl -n webapp-mysql describe pod <webapp-pod>
kubectl -n webapp-mysql top pods --containers
kubectl -n webapp-mysql logs <webapp-pod> -c webapp --previous --tail=100
kubectl -n webapp-mysql logs <webapp-pod> -c webapp --tail=100
```

Increasing restart counts and `Last State: Terminated` with `Reason: OOMKilled`
and exit code `137` identify a memory failure. Exit code `137` alone is not proof
of an out-of-memory failure. Compare memory usage with the container limit in
`describe pod`; `top` is a current sample, so usage can be low immediately after
a restart. Previous logs show the terminated container's last startup attempt.
They can stop abruptly before the HTTP server finishes starting.

After changing resources, verify the rollout and HTTP response:

```sh
kubectl -n webapp-mysql rollout status deployment/webapp-deployment --timeout=300s
kubectl -n webapp-mysql get service webapp-svc
kubectl -n webapp-mysql get endpointslice -l kubernetes.io/service-name=webapp-svc
curl -I --connect-timeout 5 --max-time 15 http://<external-ip>/
```

The Service exposes HTTP on port `80`, forwarding to container port `8080`.
It does not configure HTTPS. An HTTP response, including a login redirect, confirms
that the request reaches a web server. This Deployment has no readiness probe, so
`Running` or a completed rollout alone does not prove the application serves HTTP.

### Recover an existing Premium disk on UK South sandbox

An existing bound PVC keeps its disk when the StorageClass changes. If MySQL is
stuck in `ContainerCreating` with `FailedAttachVolume` because the node does not
support Premium storage, convert that disk without deleting the PVC or PV.
Use the intended cluster context and an Azure account with permission to update
the disk. From the repository root:

```sh
pv_name=$(kubectl -n webapp-mysql get pvc azure-managed-disk-pvc -o jsonpath='{.spec.volumeName}')
disk_id=$(kubectl get pv "$pv_name" -o jsonpath='{.spec.azureDisk.diskURI}')
az disk show --ids "$disk_id" --query '{sku:sku.name,state:diskState,managedBy:managedBy}' -o json
```

These manifests currently use the migrated Azure Disk volume format, so the
disk ID is in `spec.azureDisk.diskURI`. Confirm the disk is `Unattached` and
`managedBy` is null before conversion. If attached, first plan a controlled
workload stop and wait for detachment; do not convert an attached disk.

```sh
az disk update --ids "$disk_id" --sku StandardSSD_LRS
kubectl -n webapp-mysql rollout status deployment/mysql-deployment --timeout=300s
kubectl -n webapp-mysql rollout status deployment/webapp-deployment --timeout=300s
```

Azure limits storage-tier changes to twice per day. Kubernetes retries the disk
attachment automatically; the existing PVC and PV remain bound to the same disk.

StorageClass parameters are immutable. After publishing the overlay change to
`dev/sandbox`, replacing only the StorageClass is a one-time migration step:

```sh
kubectl kustomize clusters/azure/DEV-JKS/dev/uks/sandbox/aks-app-routing/applications/webapp-mysql-pv \
  | yq 'select(.kind == "StorageClass")' > /tmp/webapp-mysql-storage-class.yaml
kubectl replace --force -f /tmp/webapp-mysql-storage-class.yaml
```

This recreates only the StorageClass, preserving the bound PVC, PV, and disk.
Review other consumers before replacement. Do not force-replace the complete
application or its PVC. Refresh the Argo CD Application after migration and verify
that it reports `Synced` and `Healthy`.
