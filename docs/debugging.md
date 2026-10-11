# Debugging the AKS AGIC sandbox

This page records the read-only CLI commands used to investigate an AGIC
Deployment marked `Degraded`, a chaos-generator certificate stuck issuing, and
public Argo CD access failing while local HTTP access works, and External DNS
crashing after a managed identity change.
Repeated commands are listed once. Azure resource names and identity IDs are
parameters; fill them from your cluster configuration and Terraform outputs.
These checks inspect live state without applying or deleting resources.

## Select the context and resource identifiers

```sh
command -v az
command -v kubelogin
kubectl config current-context

CONTEXT=REPLACE_PLACEHOLDER
CLUSTER=clusters/azure/DEV-JKS/dev/uks/sandbox/aks-agic
AZURE_SUBSCRIPTION_ID=REPLACE_PLACEHOLDER
AKS_NAME=REPLACE_PLACEHOLDER
AKS_RESOURCE_GROUP=REPLACE_PLACEHOLDER
AGIC_CLIENT_ID=REPLACE_PLACEHOLDER
AGIC_PRINCIPAL_ID=REPLACE_PLACEHOLDER
AGIC_IDENTITY_NAME=REPLACE_PLACEHOLDER
AGIC_IDENTITY_RESOURCE_GROUP=REPLACE_PLACEHOLDER
APPGW_NAME=REPLACE_PLACEHOLDER
APPGW_RESOURCE_GROUP=REPLACE_PLACEHOLDER
APPGW_VNET_NAME=REPLACE_PLACEHOLDER
APPGW_SUBNET_NAME=REPLACE_PLACEHOLDER
ARGOCD_HOST=argocd.jkslabs.site
```

Confirm the selected context belongs to the intended AKS cluster. `AGIC_CLIENT_ID`
must match `armAuth.identityClientID` in
`$CLUSTER/platform/ingress-agic/application-patches.yaml`. The identity, AKS, and gateway
can reside in different resource groups. Azure CLI authentication must work and
its local cache must be writable. If kubectl reports `executable kubelogin not
found`, install the AKS authentication helper before checking cluster resources.

## Fix root repository authentication

If target-state generation fails with
`SSH agent requested but SSH_AUTH_SOCK not-specified`, inspect the root source URL:

```sh
kubectl --context "$CONTEXT" -n argocd get application all-apps \
  -o jsonpath='{.spec.source.repoURL}'
```

The public GitOps repository uses
`https://github.com/RyanJKS/platform-gitops.git`. Its child Applications already
use HTTPS. Set `platform_gitops_github_url` in infrastructure `repo-urls.hcl` to
that URL and plan/apply only `argocd_application_manifest`. Refresh the root
Application after the update. Runner SSH credentials do not authenticate the
Argo CD repo-server. Do not set a dummy `SSH_AUTH_SOCK` to suppress this error.
See [repository bootstrap access](bootstrap.md#repository-access).

## Separate sync state from resource health

```sh
kubectl --context "$CONTEXT" -n argocd get applications \
  -o custom-columns='NAME:.metadata.name,SYNC:.status.sync.status,HEALTH:.status.health.status,OPERATION:.status.operationState.phase,MESSAGE:.status.operationState.message' \
  --request-timeout=15s
```

`Synced` means the deployed manifests match the desired configuration. It does
not mean Pods or certificates are ready. A `Succeeded` operation can coexist with
`Degraded` or `Progressing` health. In this investigation, AGIC was `Synced` and
`Degraded`; chaos-generator was `Synced` and `Progressing`. Repeated syncs would
not resolve their runtime failures.

## Check an Argo CD port-forward connection reset

These commands were used when port forwarding `argocd-server` reported
`connection reset by peer` and `lost connection to pod`:

```sh
kubectl --context "$CONTEXT" -n argocd get deployments,pods \
  -l app.kubernetes.io/name=argocd-server -o wide --request-timeout=15s

kubectl --context "$CONTEXT" -n argocd get configmap argocd-cmd-params-cm \
  -o yaml --request-timeout=15s

kubectl --context "$CONTEXT" -n argocd logs deployment/argocd-server \
  --tail=50 --request-timeout=15s
```

The server was ready with zero restarts. Its ConfigMap contained
`server.insecure: "true"`, and its startup logs confirmed `tls: false`. This is
the AGIC bootstrap's intended backend setting: Application Gateway terminates
public TLS, while the Argo CD Pod serves HTTP.

Service ports 80 and 443 both target Pod port 8080. Forwarding Service port 443
does not enable TLS on a server configured for HTTP. Stop any old forwarding
process, reconnect, and use an explicit HTTP URL:

```sh
kubectl --context "$CONTEXT" -n argocd port-forward \
  service/argocd-server 8080:80
```

Open `http://127.0.0.1:8080`. Port forwarding can also disconnect during a Pod
replacement; a persistent reset while using HTTP requires further investigation
of the Pod and forwarding connection. The reset message alone does not establish
that the server has crashed. See [Argo CD ingress](bootstrap.md#seed-the-root-and-access-argo-cd)
for public HTTPS access and the startup parameter change.

### Compare HTTP and HTTPS through a fresh forward

To isolate the protocol from browser settings, a temporary forward was opened
on a separate local port:

```sh
kubectl --context "$CONTEXT" -n argocd port-forward \
  --address 127.0.0.1 svc/argocd-server 18080:80
```

In another terminal, the following requests were used to compare protocols:

```sh
curl --max-time 10 --silent --show-error \
  --output /tmp/argocd-portforward-http-response.html \
  --write-out 'HTTP %{http_code}\n' http://127.0.0.1:18080/

curl --max-time 10 --silent --show-error \
  --output /tmp/argocd-portforward-https-response.txt \
  --write-out 'HTTPS %{http_code}\n' https://127.0.0.1:18080/
```

HTTP returned `200`. HTTPS failed with `unexpected eof while reading`, and the
forwarding process exited. The temporary forward was stopped after the check.
This reproduced a protocol failure against the healthy HTTP backend; it does
not establish which URL a browser used. Use the explicit HTTP URL for local
access, and check browser HTTPS upgrades or cached redirects if curl succeeds
while the browser connection fails.

## Inspect the AGIC Deployment, Pod, and service account

```sh
kubectl --context "$CONTEXT" -n ingress-agic get deployments,pods \
  --request-timeout=15s

kubectl --context "$CONTEXT" -n ingress-agic get deployment ingress-agic \
  -o yaml --request-timeout=15s

kubectl --context "$CONTEXT" -n ingress-agic get serviceaccount ingress-agic \
  -o yaml --request-timeout=15s

kubectl --context "$CONTEXT" -n ingress-agic logs deployment/ingress-agic \
  --previous --tail=120 --request-timeout=15s
```

The deployed controller uses namespace `ingress-agic` and service account
`ingress-agic`. Check the Deployment's `serviceAccountName`, Pod template label
`azure.workload.identity/use: "true"`, and service account annotation
`azure.workload.identity/client-id`. The annotation must contain the configured
managed identity client ID.

`--previous` reads the last terminated container, which is useful during
`CrashLoopBackOff`. If no terminated container exists, omit `--previous` to read
the current container. The crash in this investigation contained
`WorkloadIdentityCredential authentication failed`, `AADSTS700213`, and an Azure
401. This identified a federated identity mismatch before gateway access.

## Compare the managed identity and AKS OIDC configuration

```sh
az identity list --subscription "$AZURE_SUBSCRIPTION_ID" \
  --query "[?clientId=='${AGIC_CLIENT_ID}'].{name:name,resourceGroup:resourceGroup,clientId:clientId,id:id}" \
  -o json

az aks list --subscription "$AZURE_SUBSCRIPTION_ID" \
  --query "[?name=='${AKS_NAME}'].{name:name,resourceGroup:resourceGroup,issuer:oidcIssuerProfile.issuerUrl,workloadIdentity:securityProfile.workloadIdentity.enabled}" \
  -o json

az identity federated-credential list \
  --subscription "$AZURE_SUBSCRIPTION_ID" \
  --resource-group "$AGIC_IDENTITY_RESOURCE_GROUP" \
  --identity-name "$AGIC_IDENTITY_NAME" \
  --query '[].{name:name,issuer:issuer,subject:subject,audiences:audiences}' \
  -o json
```

The identity query must return the identity referenced by AGIC. Use its returned
name and resource group in the federated credential query. The AKS query must
show an OIDC issuer and workload identity enabled. Compare these fields exactly:

| Federated credential field | Expected value |
| --- | --- |
| `issuer` | The AKS `oidcIssuerProfile.issuerUrl`, including its trailing slash |
| `subject` | `system:serviceaccount:ingress-agic:ingress-agic` |
| `audiences` | Contains `api://AzureADTokenExchange` |

The actual credential trusted `system:serviceaccount:agic:agic`, while the
controller ran as `system:serviceaccount:ingress-agic:ingress-agic`. Its issuer
and audience already matched. The following check confirmed that the old
namespace had no deployed resources:

```sh
kubectl --context "$CONTEXT" -n agic get deployments,pods,serviceaccounts \
  --request-timeout=15s
```

Correct the existing AGIC federated credential in the owning Terraform stack:

```hcl
subject = "system:serviceaccount:ingress-agic:ingress-agic"
```

Apply that infrastructure change before expecting AGIC to authenticate. After
token acquisition works, any subsequent Azure 403 should be investigated as a
role assignment or resource scope problem. The required permissions are listed
in [AGIC prerequisites](bootstrap.md#prepare-the-aks-agic-sandbox).

## Verify the Application Gateway target

```sh
az network application-gateway show \
  --subscription "$AZURE_SUBSCRIPTION_ID" \
  --resource-group "$APPGW_RESOURCE_GROUP" \
  --name "$APPGW_NAME" \
  --query '{name:name,id:id,provisioningState:provisioningState}' \
  -o json

az network application-gateway list \
  --subscription "$AZURE_SUBSCRIPTION_ID" \
  --query '[].{name:name,resourceGroup:resourceGroup}' \
  -o json
```

The target must match `appgw.subscriptionId`, `appgw.resourceGroup`, and
`appgw.name` in the cluster's AGIC values. The `show` command returned
`ResourceGroupNotFound`: the configured resource group ended in `-RGG`, while
the gateway list showed `-RG`. The repository values were corrected to the
existing gateway's resource group. Publish that correction to the Git revision
read by Argo CD and sync `ingress-agic` so it reaches the live Deployment.

## Check Ingress addresses and External DNS

```sh
kubectl --context "$CONTEXT" get ingress -A -o wide \
  --request-timeout=15s

kubectl --context "$CONTEXT" -n external-dns logs deployment/external-dns \
  --tail=50 --request-timeout=15s
```

AGIC must reconcile the selected Ingresses and populate their addresses.
External DNS uses those addresses to publish records for hosts allowed by its
domain filter. The workload and HTTP-01 solver Ingresses had empty `ADDRESS`
fields because AGIC could not start.

External DNS reported `All records are already up to date`. That message alone
does not establish that the desired hostname exists: an Ingress without an
address can produce no DNS endpoint to publish. Check the Ingress address before
treating an empty DNS update as success.

## Check External DNS identity after a ServiceAccount update

Use these commands if External DNS enters `CrashLoopBackOff` after replacing its
Azure managed identity. Set the Pod name from the deployment and Pod listing:

```sh
kubectl --context "$CONTEXT" -n external-dns get deployments,pods \
  --request-timeout=15s

EXTERNAL_DNS_POD=REPLACE_PLACEHOLDER

kubectl --context "$CONTEXT" -n external-dns describe pod "$EXTERNAL_DNS_POD" \
  --request-timeout=15s

kubectl --context "$CONTEXT" -n external-dns logs deployment/external-dns \
  --tail=80 --request-timeout=15s

kubectl --context "$CONTEXT" -n external-dns get serviceaccount external-dns \
  -o yaml --request-timeout=15s

kubectl --context "$CONTEXT" -n external-dns get deployment external-dns \
  -o jsonpath='{.spec.template.spec.containers[0].env}' --request-timeout=15s
```

Compare the Pod's injected `AZURE_CLIENT_ID` with the ServiceAccount annotation
`azure.workload.identity/client-id` and the cluster's External DNS Application patch under `helm.valuesObject`.
The workload identity webhook injects environment variables when a Pod is
created; they need not appear in the Deployment template. Updating a
ServiceAccount annotation does not rewrite existing Pod environment variables.
Restarting a crashed container within the same Pod keeps its previous variables.

The investigated Pod used an older client ID, while the live ServiceAccount and
repository values already referenced the new identity. Its logs reported
`AADSTS700016`: Azure could not find the application identifier in the configured
tenant. To compare the old and configured identities in the cluster subscription:

```sh
EXTERNAL_DNS_POD_CLIENT_ID=REPLACE_PLACEHOLDER
EXTERNAL_DNS_CLIENT_ID=REPLACE_PLACEHOLDER

az identity list --subscription "$AZURE_SUBSCRIPTION_ID" \
  --query "[?clientId=='${EXTERNAL_DNS_POD_CLIENT_ID}' || clientId=='${EXTERNAL_DNS_CLIENT_ID}'].{name:name,resourceGroup:resourceGroup,clientId:clientId,tenantId:tenantId,principalId:principalId}" \
  -o json
```

The query found the new identity in the expected tenant; it did not return the
old identity in this subscription. The immediate failure was the stale Pod's
client ID, not a DNS record or health probe error.

After confirming the ServiceAccount annotation is correct, recreate the Pod by
restarting the Deployment. These recovery commands change live state and were
not run during this inspection:

```sh
kubectl --context "$CONTEXT" -n external-dns rollout restart deployment/external-dns
kubectl --context "$CONTEXT" -n external-dns rollout status deployment/external-dns \
  --timeout=120s
```

Inspect the replacement Pod and its logs. The new Pod should receive the updated
client ID. If authentication still fails, verify the identity's federated
credential trusts `system:serviceaccount:external-dns:external-dns`, matches the
current AKS OIDC issuer, and permits `api://AzureADTokenExchange`. Check Azure DNS
role assignments only after token acquisition succeeds.

## Check public Argo CD access

Run these checks when local port forwarding works but the public hostname does
not. Re-run the AGIC Deployment and Pod check above to confirm current health.
Controller readiness and an Ingress address alone do not prove the gateway has
listeners and reachable backends.

### Inspect DNS and the public HTTPS endpoint

```sh
kubectl --context "$CONTEXT" -n argocd get ingress,certificate \
  --request-timeout=15s

kubectl --context "$CONTEXT" -n external-dns logs deployment/external-dns \
  --tail=30 --request-timeout=15s

getent ahostsv4 "$ARGOCD_HOST"

curl --max-time 10 --head --silent --show-error "https://${ARGOCD_HOST}/"
```

Compare the resolved IPv4 address with the Ingress `ADDRESS` and gateway public
IP. `getent` checks this machine's resolver; External DNS logs show whether the
controller published the record. A request timeout requires further gateway and
network checks. A certificate trust error instead requires checking the issued
certificate and issuer; this bootstrap now selects Let's Encrypt production.

### Inspect the Argo CD certificate and HTTP-01 route

```sh
kubectl --context "$CONTEXT" -n argocd describe certificate argocd-ingress-tls \
  --request-timeout=15s

kubectl --context "$CONTEXT" -n argocd get challenges.acme.cert-manager.io \
  -o custom-columns='NAME:.metadata.name,STATE:.status.state,DNS:.spec.dnsName,PRESENTED:.status.presented,REASON:.status.reason' \
  --request-timeout=15s

kubectl --context "$CONTEXT" -n argocd get ingress argocd-server \
  -o yaml --request-timeout=15s

kubectl --context "$CONTEXT" -n argocd get endpoints --request-timeout=15s
```

The live Ingress should route `/` to `argocd-server:80`. During validation,
cert-manager's edit-in-place solver adds the exact ACME challenge path and its
solver Service. Confirm both Services have endpoints. The Endpoints command above
was used during diagnosis; Kubernetes 1.33 and later deprecate this API. For new
checks, use EndpointSlices instead:

```sh
kubectl --context "$CONTEXT" -n argocd get endpointslices.discovery.k8s.io \
  -o wide --request-timeout=15s
```

`presented=true` means the solver has been configured; it does not mean the
challenge URL is reachable. A self-check reporting `wrong status code '502',
expected '200'` means the HTTP response failed validation. Inspect gateway routing
and backend health before retrying certificate issuance.

### Inspect gateway backend health and AKS networking

```sh
az network application-gateway show-backend-health \
  --subscription "$AZURE_SUBSCRIPTION_ID" \
  --resource-group "$APPGW_RESOURCE_GROUP" --name "$APPGW_NAME" \
  --query 'backendAddressPools[].{pool:backendAddressPool.id,settings:backendHttpSettingsCollection[].{httpSettings:backendHttpSettings.id,servers:servers}}' \
  -o json

az aks show --subscription "$AZURE_SUBSCRIPTION_ID" \
  --resource-group "$AKS_RESOURCE_GROUP" --name "$AKS_NAME" \
  --query '{network:networkProfile,agentPools:agentPoolProfiles[].{name:name,vnetSubnetId:vnetSubnetId,podSubnetId:podSubnetId}}' \
  -o json
```

Check whether the gateway contains the pools expected for the Ingress backends
and whether their servers are healthy. An empty bootstrap pool requires checking
AGIC's target and reconciliation logs before assuming an application failure.
Use the network profile to identify the AKS network mode and subnet layout when
investigating backend connectivity.

### Confirm AGIC gateway update permissions

If AGIC is ready but the gateway retains only bootstrap configuration, inspect
the current container logs rather than using `--previous`:

```sh
kubectl --context "$CONTEXT" -n ingress-agic logs deployment/ingress-agic \
  --tail=160 --request-timeout=15s

kubectl --context "$CONTEXT" -n ingress-agic get configmap ingress-agic \
  -o yaml --request-timeout=15s

az network application-gateway show \
  --subscription "$AZURE_SUBSCRIPTION_ID" \
  --resource-group "$APPGW_RESOURCE_GROUP" --name "$APPGW_NAME" \
  --query '{state:provisioningState,operationalState:operationalState,subnets:gatewayIPConfigurations[].subnet.id,listeners:httpListeners[].{name:name,host:hostName,port:frontendPort.id},pools:backendAddressPools[].{name:name,addresses:backendAddresses},rules:requestRoutingRules[].{name:name,type:ruleType},urlMaps:urlPathMaps[].name}' \
  -o json

az identity show --subscription "$AZURE_SUBSCRIPTION_ID" \
  --resource-group "$AGIC_IDENTITY_RESOURCE_GROUP" \
  --name "$AGIC_IDENTITY_NAME" \
  --query '{name:name,clientId:clientId,principalId:principalId}' -o json
```

Set `AGIC_PRINCIPAL_ID` to the returned `principalId`, which is the identity's
object ID, not its client ID. Compare the log's rejected object ID with that
value. Then inspect the identity's assignments in this subscription:

```sh
az role assignment list --subscription "$AZURE_SUBSCRIPTION_ID" --all \
  --query "[?principalId=='${AGIC_PRINCIPAL_ID}'].{role:roleDefinitionName,scope:scope}" \
  -o json
```

The current AGIC log showed `ApplicationGatewayInsufficientPermissionOnSubnet`:
Azure rejected the gateway `CreateOrUpdate` because the AGIC identity lacked
`Microsoft.Network/virtualNetworks/subnets/join/action` on the gateway subnet
`agwsnet-001`. The rejected object ID matched the configured AGIC managed
identity. Its assignments were Reader on the resource group, Contributor on the
gateway, and Managed Identity Operator on its own identity. None granted the
required subnet action.

AGIC had generated routes for Argo CD, chaos-generator, and their ACME solvers,
but Azure rejected the update. The live gateway remained `Running` with
`Succeeded` provisioning state and only the bootstrap listener, rule, and empty
pool. Those gateway states describe the retained configuration and do not prove
AGIC's attempted update succeeded.

In the owning Terraform stack, grant the AGIC identity Network Contributor on
the actual gateway subnet, or a custom role that permits the required subnet
join action. Scope the assignment to the subnet returned by the gateway query;
do not use the AKS node subnet or infer it from `APPGW_SUBNET_NAME`. The live
ConfigMap's subnet name differed from the actual gateway subnet. Use the AGIC
principal object ID for the role assignment. Contributor on the gateway and
Reader on its resource group alone are insufficient for this update.

After the infrastructure correction propagates, verify a subsequent AGIC update
succeeds and repeat the backend health, public endpoint, and certificate checks.
This investigation confirmed the permission blocker; backend connectivity and
certificate readiness still require verification after it is removed.

### Inspect the served certificate and an HSTS browser error

When Brave reports incorrect credentials and prevents access because the site
uses HSTS, inspect the public certificate without changing browser security:

```sh
timeout 15s openssl s_client -connect "${ARGOCD_HOST}:443" \
  -servername "$ARGOCD_HOST" </dev/null 2>/dev/null \
  | openssl x509 -noout -subject -issuer -dates -ext subjectAltName
```

The follow-up inspection received a certificate for `argocd.jkslabs.site` issued
by `cert-manager.local`. This was cert-manager's temporary certificate, not a
publicly trusted certificate. The Certificate remained `Ready=False`, referenced
`letsencrypt-staging`, and its HTTP-01 challenge still received `502`.

HSTS requires trusted HTTPS and prevents bypassing this certificate error. The
observed temporary certificate explains the browser warning; the warning alone
does not establish an attack. Resolve validation and issue a certificate through
a production issuer for normal browser access. Completing staging issuance alone
will still produce an untrusted certificate. Use the local HTTP port forward
while resolving public TLS.

### Check AGIC compatibility with Azure CNI overlay

At the same follow-up, gateway backend health contained the Argo CD,
chaos-generator, and ACME solver pools. Every backend was `Unhealthy`; probe logs
reported that the response exceeded the timeout threshold. The earlier rejected
update was no longer the only observed state: application pools now existed,
but their backends were not healthy.

```sh
az network application-gateway show-backend-health \
  --subscription "$AZURE_SUBSCRIPTION_ID" \
  --resource-group "$APPGW_RESOURCE_GROUP" --name "$APPGW_NAME" \
  --query 'backendAddressPools[0].backendHttpSettingsCollection[0].servers[0].healthProbeLog' \
  -o tsv

kubectl --context "$CONTEXT" -n ingress-agic get deployment ingress-agic \
  -o jsonpath='{.spec.template.spec.containers[*].image}' --request-timeout=15s

az network vnet subnet show --subscription "$AZURE_SUBSCRIPTION_ID" \
  --resource-group "$APPGW_RESOURCE_GROUP" --vnet-name "$APPGW_VNET_NAME" \
  --name "$APPGW_SUBNET_NAME" \
  --query '{addressPrefix:addressPrefix,addressPrefixes:addressPrefixes,delegations:delegations[].serviceName}' \
  -o json
```

Set the subnet and virtual network names from the actual gateway subnet resource
ID, not the controller's ConfigMap. If the virtual network belongs to another
resource group, use that resource group for the subnet query.

The live controller image was `kubernetes-ingress:1.8.1`. The AKS networking check
above identified Azure CNI overlay. Microsoft's
[AGIC networking requirements](https://learn.microsoft.com/en-us/azure/application-gateway/ingress-controller-overview#container-networking-and-agic)
require AGIC `v1.9.1` or later for Azure CNI overlay, a gateway subnet of `/24`
or smaller, and delegation to `Microsoft.Network/applicationGateways`. The live
gateway subnet was `/24` and already had that delegation, but the controller
version did not meet the requirement.

The catalog now pins the AGIC Helm chart to `1.9.1`. Publish the change and sync
the controller before repeating backend and certificate checks. The observed
`1.8.1` image above describes the earlier live inspection; this repository change
does not establish that the live controller has upgraded. Follow the
[AGIC upgrade procedure](#confirm-the-chart-upgrade-reached-the-live-cluster).
Investigate remaining networking or probe failures if backends stay unhealthy
after the upgrade. Switch the certificate configuration to a production issuer
after successful staging validation.

### Confirm the chart upgrade reached the live cluster

Compare the live Application's chart version with its Deployment image and the
catalog pin. Use the image query above and these Application checks:

```sh
kubectl --context "$CONTEXT" -n argocd get application ingress-agic \
  -o custom-columns='NAME:.metadata.name,CHART:.spec.sources[0].targetRevision,SYNC:.status.sync.status,HEALTH:.status.health.status,OPERATION:.status.operationState.phase' \
  --request-timeout=15s

kubectl --context "$CONTEXT" -n argocd get application all-apps \
  -o custom-columns='NAME:.metadata.name,BRANCH:.spec.source.targetRevision,PATH:.spec.source.path,SYNC:.status.sync.status,COMMIT:.status.sync.revision' \
  --request-timeout=15s

kubectl --context "$CONTEXT" -n argocd get certificate argocd-ingress-tls \
  -o yaml --request-timeout=15s
```

To distinguish a certificate trust error from a failed application route, the
following diagnostic request bypasses certificate verification for this one
request and discards the response body. This is not a browser access workaround;
do not use it to submit credentials:

```sh
curl --max-time 10 --silent --show-error --insecure --output /dev/null \
  --write-out 'HTTP %{http_code} remote_ip %{remote_ip} certificate_verification %{ssl_verify_result}\n' \
  "https://${ARGOCD_HOST}/"
```

At the October 10 follow-up, the live `ingress-agic` Application still selected
`1.8.1`, and its Deployment still ran the `1.8.1` image. Argo CD reported `Synced`,
`Healthy`, and a succeeded operation for that version. The root was synced to
`dev/sandbox`, while the local catalog change selected `1.9.1`. A local file edit
does not change the Git revision read by Argo CD or deploy the new controller.

The public hostname reached the gateway, but the diagnostic request returned
`502`. All application and ACME solver backends were unhealthy with probe
timeouts. The certificate was still temporary, issued by `cert-manager.local`,
and the staging HTTP-01 challenge was pending with a `502` self-check failure.
External DNS had recovered and its replacement Pod was ready with zero restarts.

Publish the catalog change to `dev/sandbox`, refresh and reconcile `all-apps`,
then confirm the child Application selects `1.9.1` and sync `ingress-agic`.
Verify the new Deployment image and backend health before expecting certificate
issuance to succeed. An Application's `Healthy` status reflects its managed
Kubernetes resources; it does not prove the gateway can reach workload backends.
Successful staging validation still needs a subsequent production issuer change
for browser-trusted HTTPS.

### Check overlay setup after a successful AGIC upgrade

At the next inspection, the live Application selected `1.9.1`, its sync had
succeeded, and the controller Pod ran `1.9.1` with zero restarts. All six
application and ACME solver backends still timed out. Read the current Pod's
startup logs, since a short tail can omit the networking error:

```sh
kubectl --context "$CONTEXT" -n ingress-agic get deployments,pods -o wide \
  --request-timeout=15s

kubectl --context "$CONTEXT" -n ingress-agic logs deployment/ingress-agic \
  --request-timeout=15s

az aks show --subscription "$AZURE_SUBSCRIPTION_ID" \
  --resource-group "$AKS_RESOURCE_GROUP" --name "$AKS_NAME" \
  --query '{network:networkProfile,nodeResourceGroup:nodeResourceGroup,agentPools:agentPoolProfiles[].{name:name,subnet:vnetSubnetId}}' \
  -o json

kubectl --context "$CONTEXT" get customresourcedefinition \
  nodenetworkconfigs.acn.azure.com overlayextensionconfigs.acn.azure.com \
  --request-timeout=15s

kubectl --context "$CONTEXT" -n kube-system get nodenetworkconfigs.acn.azure.com \
  -o custom-columns='NAME:.metadata.name,LABELS:.metadata.labels' \
  --request-timeout=15s

kubectl --context "$CONTEXT" get overlayextensionconfigs.acn.azure.com -A \
  -o yaml --request-timeout=15s
```

AKS still reported `networkPlugin: azure` and `networkPluginMode: overlay`, and
the node networking resources had `kubernetes.azure.com/podnetwork-type: overlay`.
Both networking CRDs existed, but no OverlayExtensionConfig had been created.
The controller log reported `AuthorizationFailed` with HTTP `403` for
`Microsoft.Network/routeTables/read` in the AKS managed node resource group.

Despite the log saying `failed to reconcile kubenet CNI`, this did not establish
that the cluster used kubenet. The
[AGIC 1.9.1 reconciler](https://github.com/Azure/application-gateway-kubernetes-ingress/blob/1.9.1/pkg/cni/cni.go)
attempts route-table reconciliation before overlay reconciliation. The route-table
authorization error returns before it can create the overlay extension required
for gateway access to overlay Pods.

Set the node resource group from the AKS query and the route-table name from the
controller error. Check the referenced resource and the gateway network controls:

```sh
AKS_NODE_RESOURCE_GROUP=REPLACE_PLACEHOLDER
AKS_ROUTE_TABLE_NAME=REPLACE_PLACEHOLDER

az network route-table show --subscription "$AZURE_SUBSCRIPTION_ID" \
  --resource-group "$AKS_NODE_RESOURCE_GROUP" --name "$AKS_ROUTE_TABLE_NAME" \
  --query '{id:id,routes:routes[].{prefix:addressPrefix,type:nextHopType,nextHop:nextHopIpAddress}}' \
  -o json

az network vnet subnet show --subscription "$AZURE_SUBSCRIPTION_ID" \
  --resource-group "$APPGW_RESOURCE_GROUP" --vnet-name "$APPGW_VNET_NAME" \
  --name "$APPGW_SUBNET_NAME" \
  --query '{prefixes:addressPrefixes,delegations:delegations[].serviceName,nsg:networkSecurityGroup.id,routeTable:routeTable.id}' \
  -o json

az network application-gateway show --subscription "$AZURE_SUBSCRIPTION_ID" \
  --resource-group "$APPGW_RESOURCE_GROUP" --name "$APPGW_NAME" \
  --query '{state:provisioningState,subnets:gatewayIPConfigurations[].subnet.id,tags:tags}' \
  -o json

APPGW_NSG_NAME=REPLACE_PLACEHOLDER

az network nsg show --subscription "$AZURE_SUBSCRIPTION_ID" \
  --resource-group "$APPGW_RESOURCE_GROUP" --name "$APPGW_NSG_NAME" \
  --query '{rules:securityRules[].{name:name,priority:priority,direction:direction,access:access,protocol:protocol,source:sourceAddressPrefix,sources:sourceAddressPrefixes,destination:destinationAddressPrefix,destinations:destinationAddressPrefixes,port:destinationPortRange,ports:destinationPortRanges}}' \
  -o json
```

Re-run the identity role-assignment query above with the current AGIC principal
object ID. The inspected identity had Contributor on the gateway, Network
Contributor on its subnet, and Reader on the infrastructure resource group. It
had no Reader assignment on the separate `MC_...` managed node resource group.
The route-table query using the operator's account returned `ResourceNotFound`;
AGIC lacked permission to establish that absence and instead received `403`.
The gateway subnet had no route table attached and retained the required overlay
delegation.

Microsoft's [standard manual installation guide](https://learn.microsoft.com/en-us/azure/application-gateway/ingress-controller-install-existing)
lists Contributor on the gateway and Reader on its resource group, with
additional network resource permissions. It does not prescribe blanket Reader
access to the AKS managed node resource group for every deployment. The managed
group recommendation here comes from the observed error and the `1.9.1` code
path. The later Reader assignment resolved this deployment's networking failure,
as recorded below. The narrower custom-role example remains untested live.

For that workaround, permit `Microsoft.Network/routeTables/read` at the managed
node resource group scope returned by `az aks show`, using a restricted custom
role in the owning Terraform stack. Reader on that group also covers the
operation, but grants broader read access. The
[AGIC route-table check](https://github.com/Azure/application-gateway-kubernetes-ingress/blob/1.9.1/pkg/azure/client.go)
skips a missing table when Azure returns `404`; a `403` aborts networking
reconciliation. Do not create the absent route table to resolve this overlay
failure.

If the workaround is applied, wait for permissions to propagate, check subsequent
controller logs, and
verify `agic-overlay-extension-config` exists and reports `Succeeded`, then
repeat backend health and certificate checks. Additional networking failures
still require investigation if probes remain unhealthy. Increasing probe
timeouts does not resolve this authorization failure. No role assignment or
network resource was changed during this inspection.

### Apply the scoped route-table read workaround

The reviewed AGIC `1.9.1` chart has no dedicated setting to skip its route-table
lookup. For the observed authorization failure, the proposed Terraform change
below grants only that read operation. It does not grant Reader across all
resource types in the managed group. Integrate it into the stack that owns the
AGIC identity and AKS permissions, using existing stack references where possible.
This example has not been applied to Azure or verified against live backends.

```hcl
variable "aks_node_resource_group_id" {
  type        = string
  description = "Resource ID of the MC_ resource group returned by AKS nodeResourceGroup."
}

variable "agic_principal_id" {
  type        = string
  description = "Principal object ID of the AGIC managed identity, not its client ID."
}

resource "azurerm_role_definition" "agic_route_table_read" {
  name        = "agicuksdevaks-route-table-read"
  scope       = var.aks_node_resource_group_id
  description = "Allows the AGIC route-table existence check before overlay setup."

  permissions {
    actions     = ["Microsoft.Network/routeTables/read"]
    not_actions = []
  }

  assignable_scopes = [var.aks_node_resource_group_id]
}

resource "azurerm_role_assignment" "agic_route_table_read" {
  scope              = var.aks_node_resource_group_id
  role_definition_id = azurerm_role_definition.agic_route_table_read.role_definition_resource_id
  principal_id       = var.agic_principal_id
}
```

Review and apply the change through the owning Terraform stack. Once the role
assignment propagates, inspect the overlay extension and controller logs. An
Azure role change alone does not require an Argo CD sync. If overlay setup has
already succeeded, no restart is needed. If the controller has not retried
networking reconciliation, restart it to trigger another attempt. These are
proposed recovery commands; they were not run during the investigation:

```sh
kubectl --context "$CONTEXT" -n ingress-agic rollout restart deployment/ingress-agic
kubectl --context "$CONTEXT" -n ingress-agic rollout status deployment/ingress-agic \
  --timeout=120s
kubectl --context "$CONTEXT" -n ingress-agic logs deployment/ingress-agic \
  --tail=100 --request-timeout=15s
kubectl --context "$CONTEXT" get overlayextensionconfigs.acn.azure.com -A \
  -o yaml --request-timeout=15s
```

Verify the route-table `403` stops, `agic-overlay-extension-config` reports
`Succeeded`, and gateway backends become healthy. Repeat the existing backend
health and certificate checks. If authorization or connectivity still fails,
investigate the new error before expanding permissions. Keep staging until its
issuance succeeds, then select a production issuer for browser-trusted HTTPS.

### Verify recovery after the Reader assignment

The operator added Reader for the AGIC identity on the AKS managed node resource
group. The identity role-assignment query confirmed that scope. The same AGIC
Pod then created `agic-overlay-extension-config` without an additional Argo CD
sync or controller restart during this inspection.

```sh
kubectl --context "$CONTEXT" get overlayextensionconfigs.acn.azure.com -A \
  -o yaml --request-timeout=15s

kubectl --context "$CONTEXT" -n ingress-agic get pods --request-timeout=15s

kubectl --context "$CONTEXT" -n ingress-agic logs deployment/ingress-agic \
  --since=5m --request-timeout=15s

kubectl --context "$CONTEXT" -n argocd get certificate argocd-ingress-tls \
  --request-timeout=15s

kubectl --context "$CONTEXT" -n chaos-generator get certificates \
  --request-timeout=15s
```

The extension reported `state: Succeeded` and successful programming of the
network containers for the gateway subnet. Controller logs confirmed overlay
configuration became ready. Gateway backend health showed the Argo CD backend
and all three chaos-generator backends as `Healthy`, with `200` probe responses.
The solver pools had disappeared after issuance completed.

The public HTTPS diagnostic request with certificate verification disabled
returned `200`, and both staging Certificates reported `Ready=True`. The served
certificate issuer was Let's Encrypt `(STAGING) Ersatz Emmer YR2`. This confirms
application routing and staging validation recovered; the staging certificate
remains untrusted by browsers. The next configuration step is the
[production issuer change](#configure-production-tls-after-staging-succeeds),
including the Argo CD Ingress annotation configured through shared bootstrap/self-management values. Re-syncing unchanged
staging manifests will not make that certificate publicly trusted.

### Configure production TLS after staging succeeds

After successful staging issuance, the cluster manifests select `letsencrypt-prod`
with the production ACME endpoint and `letsencrypt-prod-account-key`. The existing
contact email and AGIC solver are retained. Argo CD continues using
`argocd-ingress-tls`; chaos-generator continues using `chaos-generator-tls`.
The staging issuer and its account key are not deleted during this migration.

The rollout creates the production issuer first, then updates the two live Ingress
annotations. These commands change live state and are part of the requested
production configuration:

```sh
kubectl --context "$CONTEXT" apply --server-side \
  -k "$CLUSTER/platform/cert-manager"

kubectl --context "$CONTEXT" wait --for=condition=Ready \
  clusterissuer/letsencrypt-prod --timeout=300s

kubectl --context "$CONTEXT" -n argocd annotate ingress argocd-server \
  cert-manager.io/cluster-issuer=letsencrypt-prod --overwrite

kubectl --context "$CONTEXT" -n chaos-generator annotate ingress chaos-generator \
  cert-manager.io/cluster-issuer=letsencrypt-prod --overwrite
```

Verify the Certificates have adopted the production issuer and become ready, then
check the public endpoints with normal certificate verification:

```sh
kubectl --context "$CONTEXT" -n argocd get certificate argocd-ingress-tls \
  -o custom-columns='NAME:.metadata.name,ISSUER:.spec.issuerRef.name,READY:.status.conditions[?(@.type=="Ready")].status' \
  --request-timeout=15s

kubectl --context "$CONTEXT" -n chaos-generator get certificate chaos-generator-tls \
  -o custom-columns='NAME:.metadata.name,ISSUER:.spec.issuerRef.name,READY:.status.conditions[?(@.type=="Ready")].status' \
  --request-timeout=15s

curl --max-time 15 --head --silent --show-error "https://${ARGOCD_HOST}/"
curl --max-time 15 --head --silent --show-error https://chaos.jkslabs.site/

kubectl --context "$CONTEXT" -n chaos-generator wait --for=condition=Ready \
  certificate/chaos-generator-tls --timeout=60s

timeout 15s openssl s_client -connect chaos.jkslabs.site:443 \
  -servername chaos.jkslabs.site </dev/null 2>/dev/null \
  | openssl x509 -noout -subject -issuer -dates -ext subjectAltName
```

The requested migration was applied on October 10. Both Certificates referenced
`letsencrypt-prod` and reported `Ready=True`. Both public URLs returned HTTP `200`
with normal curl certificate verification. The gateway served publicly trusted
Let's Encrypt certificates issued by `YR1`, with the correct hostnames in their
subject alternative names. No Argo CD server restart or TLS Secret deletion was
needed.

Publish the matching Git changes to `dev/sandbox` before future child syncs.
Those Applications still use manual sync, so an older Git revision could restore
the staging annotation on a subsequent sync. The The Argo CD Ingress annotation is configured in
`platform/argocd/application-patches.yaml`. The minimal Terraform installation has no ingress; child
`argocd` consumes these runtime values after the
[ownership handoff](bootstrap.md#transfer-helm-resource-ownership-to-argo-cd).
No Argo CD server restart is needed for the certificate change.

### Public-route observations before the permission check

At the latest inspection, after the earlier AGIC startup failures:

- AGIC was ready with zero Pod restarts. The Argo CD Ingress had the gateway's
  public IP, and the public hostname resolved to that IP.
- External DNS published the `argocd` and `chaos` A records and ownership TXT
  records. The external HTTPS request still timed out after 10 seconds.
- `argocd-ingress-tls` was `Ready=False`. Its presented HTTP-01 challenge remained
  pending because the self-check received `502` instead of `200`.
- The Argo CD and ACME solver Services both had endpoints. The live Ingress
  contained the application route and the challenge route.
- Gateway backend health showed only the Terraform bootstrap pool with no
  servers; no application backend pools appeared in that response.
- AKS used Azure CNI overlay, with no separate Pod subnet. This is a networking
  detail to investigate, not a confirmed cause of the public access failure.

These observations established that DNS publication succeeded while the public
route and certificate validation still failed. The subsequent permission check
above identified the rejected gateway update. Re-run the checks after any
correction; the earlier crash and DNS failures below describe historical states.

## Inspect the workload certificate challenge

```sh
kubectl --context "$CONTEXT" -n chaos-generator \
  get challenges.acme.cert-manager.io \
  -o custom-columns='NAME:.metadata.name,STATE:.status.state,DNS:.spec.dnsName,PRESENTED:.status.presented,REASON:.status.reason' \
  --request-timeout=15s
```

Certificate events `Issuing`, `Generated`, and `Requested` describe the start of
issuance. The Challenge's `status.reason` identifies the active blocker. In the
earlier investigation, the challenge was presented but pending, and its HTTP-01
self-check failed because the workload hostname returned `no such host`. Resolve
AGIC federation and the gateway target first. Then verify AGIC readiness, Ingress
addresses, DNS publication, and successful challenge validation. Further HTTP-01
failures require checking public DNS, port 80 reachability, and gateway backend
health. See [bootstrap verification](bootstrap.md#sync-the-children-and-verify-deployment) and
[Argo CD ingress](bootstrap.md#seed-the-root-and-access-argo-cd) for the expected configuration.

The inspection and protocol-probe commands were used for diagnosis; the
port-forward on port 8080 is the recommended local reconnection step. Earlier
diagnosis was read-only. The production TLS section records the requested issuer
creation and live Ingress annotation changes. Azure role changes were applied by
the operator through the owning infrastructure setup.
