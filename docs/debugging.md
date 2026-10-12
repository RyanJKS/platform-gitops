# Debugging lookup

Use the matching problem below. Run from the repository root and check the selected cluster and subscription first.

## Setup

Replace these sandbox values when checking another cluster.

```sh
kubectl config current-context
az account show --query '{name:name,id:id}' -o table
CLUSTER=clusters/azure/DEV-JKS/dev/uks/sandbox/aks-agic
AZURE_RG=SANDBOX-SHARED-UKS-DEV-RG
AKS_NAME=agicuksdevaks
APPGW_NAME=sandboxuksdevagw
AGIC_UAMI=REPLACE_WITH_AGIC_UAMI_NAME
```

## Root application errors

**Problem:** `ComparisonError`, missing files, or missing project.

**Solution:** Fix the matching source setting, publish it, then refresh and sync `all-apps` before the child.

| Error | Fix |
| --- | --- |
| `valueObject: field not declared in schema` | Use `helm.valuesObject`. |
| Missing `values.yaml` | Update the child Application's stale `valueFiles` references. |
| Project `bootstrap` missing | Argo CD belongs to `platform-apps`; allow its chart repo, namespace, and resource kinds. |
| `SSH agent requested but SSH_AUTH_SOCK not-specified` | Use the public HTTPS repo URL or configure an Argo CD SSH credential. |

```sh
kubectl -n argocd get application all-apps \
  -o jsonpath='{range .status.conditions[*]}{.type}{": "}{.message}{"\n"}{end}'
kubectl kustomize "$CLUSTER" > /tmp/aks-agic-rendered.yaml
```

A successful render does not validate Application fields against the Argo CD CRD.

## Synced but unhealthy

**Problem:** `Synced` with `Progressing` or `Degraded`, or root sync waits on a child.

**Solution:** Check runtime health. The root waits for each child to be both `Synced` and `Healthy`. Terminate an old operation before retrying corrected manifests.

```sh
kubectl -n argocd get applications \
  -o custom-columns='NAME:.metadata.name,SYNC:.status.sync.status,HEALTH:.status.health.status,OPERATION:.status.operationState.phase,MESSAGE:.status.operationState.message'
kubectl get pods -A
```

## ClusterIssuer CRD missing

**Problem:** Kubernetes cannot find `cert-manager.io/ClusterIssuer`.

**Solution:** Install cert-manager first with `crds.enabled: true`. A parent-managed issuer needs `SkipDryRunOnMissingResource=true` while the child installs CRDs.

```sh
kubectl get crd clusterissuers.cert-manager.io
kubectl -n argocd get application cert-manager
kubectl get clusterissuer letsencrypt-prod
```

## cert-manager sync never finishes

**Problem:** Sync waits on `cert-manager-startupapicheck`.

**Solution:** Look for `FailedCreate` or a missing ServiceAccount. Create the Job's account and RBAC before it: dependencies in child wave `0`, Job in wave `1`. Keep dependencies until the Job completes, then retry a full sync.

```sh
kubectl -n cert-manager describe job cert-manager-startupapicheck
kubectl -n cert-manager get serviceaccounts,roles,rolebindings
```

## cert-manager webhook stays OutOfSync

**Problem:** Healthy cert-manager repeatedly differs on `cert-manager-webhook`.

**Solution:** AKS adds namespace exclusions. Use the cluster Application's narrow `ignoreDifferences` rule and `RespectIgnoreDifferences=true`. Match only `control-plane NotIn ["true"]` and `kubernetes.azure.com/managedby NotIn ["aks"]`.

```sh
kubectl get validatingwebhookconfiguration cert-manager-webhook \
  -o jsonpath='{.webhooks[*].namespaceSelector}{"\n"}'
kubectl -n argocd get application cert-manager -o jsonpath='{.spec.ignoreDifferences}{"\n"}'
```

Publish, refresh, and sync the root. Other webhook settings must remain managed.

## Local Argo CD connection resets

**Problem:** Port-forward fails with TLS or connection-reset errors.

**Solution:** With `server.insecure: true`, use HTTP locally. Application Gateway still serves public HTTPS.

```sh
kubectl -n argocd get cm argocd-cmd-params-cm -o jsonpath='{.data.server\.insecure}{"\n"}'
kubectl -n argocd port-forward service/argocd-server 8080:80
```

Open `http://localhost:8080`.

## SSO rejects redirect URL

**Problem:** `Invalid redirect URL: the protocol and host (including port) must match`.

**Solution:** Use `https://argocd.jkslabs.site`, or allow the exact browser origin in `configs.cm.additionalUrls` (for example, `http://localhost:8080`). Register the matching `/auth/callback` URL in Entra.

```sh
kubectl -n argocd get cm argocd-cm \
  -o jsonpath='{.data.url}{"\n"}{.data.additionalUrls}{"\n"}'
```

If Git contains the allowance but this output does not, update the root's `Application/argocd` resource, then sync the Argo CD child.

## AGIC identity errors

**Problem:** `AADSTS700213`, `AADSTS700016`, or Kubernetes `forbidden`.

**Solution:** Match the UAMI client ID, AKS issuer, ServiceAccount subject, and audience. Keep `armAuth.type: workloadIdentity` and `rbac.enabled: true` in AGIC's complete `valuesObject`.

```sh
kubectl -n ingress-agic get deployment ingress-agic \
  -o jsonpath='{.spec.template.spec.serviceAccountName}{"\n"}'
kubectl -n ingress-agic get serviceaccount ingress-agic -o yaml
az aks show -g "$AZURE_RG" -n "$AKS_NAME" --query oidcIssuerProfile.issuerUrl -o tsv
az identity federated-credential list -g "$AZURE_RG" --identity-name "$AGIC_UAMI" \
  --query '[].{subject:subject,issuer:issuer,audience:audiences}' -o json
```

Expected subject: `system:serviceaccount:ingress-agic:SERVICEACCOUNT_NAME`.
Audience: `api://AzureADTokenExchange`. Issuer must match exactly, including its trailing slash.
Chart `1.9.1` derives the account name from `fullnameOverride`; this also affects other resource names.

## AGIC Azure permissions

**Problem:** AGIC logs `AuthorizationFailed` or `ApplicationGatewayInsufficientPermissionOnSubnet`.

**Solution:** Grant the missing permission at its actual resource scope through infrastructure.

| Missing access | Role and scope |
| --- | --- |
| Gateway updates | Contributor on the Application Gateway. |
| Gateway subnet join/update | Network Contributor on the gateway subnet. |
| `Microsoft.Network/routeTables/read` | Reader, or a custom read role, on the AKS `MC_...` node resource group. |

```sh
kubectl -n ingress-agic logs deployment/ingress-agic --since=30m \
  | rg 'AuthorizationFailed|InsufficientPermission|routeTables/read'
AGIC_PRINCIPAL_ID=$(az identity show -g "$AZURE_RG" -n "$AGIC_UAMI" --query principalId -o tsv)
az role assignment list --all \
  --query "[?principalId=='$AGIC_PRINCIPAL_ID'].{role:roleDefinitionName,scope:scope}" -o table
az aks show -g "$AZURE_RG" -n "$AKS_NAME" --query nodeResourceGroup -o tsv
```

Reader on the shared resource group does not cover the separate node resource group.
Terraform exposes that scope as `azurerm_kubernetes_cluster.this.node_resource_group_id`; assign the UAMI's `principal_id`, not its `client_id`.

After role propagation, restart AGIC if networking has not retried:

```sh
kubectl -n ingress-agic rollout restart deployment/ingress-agic
kubectl -n ingress-agic rollout status deployment/ingress-agic --timeout=120s
```

## Public URL returns 502

**Problem:** Gateway backends or ACME solver probes time out.

**Solution:** Check gateway-to-Pod connectivity and AGIC network errors. For Azure CNI overlay, use AGIC `1.9.1+` and expect the overlay extension to report `Succeeded`. Fix permission errors above before changing probe timeouts.

```sh
az network application-gateway show-backend-health -g "$AZURE_RG" -n "$APPGW_NAME" \
  --query 'backendAddressPools[].backendHttpSettingsCollection[].servers[].{address:address,health:health,reason:healthProbeLog}' -o table
kubectl -n ingress-agic get deployment ingress-agic \
  -o jsonpath='{.spec.template.spec.containers[0].image}{"\n"}'
kubectl get overlayextensionconfigs.acn.azure.com -A -o yaml
kubectl -n argocd get pods,services,endpointslices -l acme.cert-manager.io/http01-solver=true
```

Look for ready solver endpoints, healthy gateway backends, and HTTP `200` on the challenge path. If probes still time out, inspect subnet delegation, NSGs, and routes.

## DNS record missing

**Problem:** Hostname returns `no such host` or has no A record.

**Solution:** Confirm the Ingress publishes a gateway address, then check External DNS identity, zone permissions, and domain filters.

```sh
kubectl get ingress -A
dig +short argocd.jkslabs.site
kubectl -n external-dns logs deployment/external-dns --tail=50
```

## External DNS keeps old identity

**Problem:** External DNS reports `AADSTS700016` after changing its UAMI.

**Solution:** Match the ServiceAccount annotation to the Pod's injected client ID. Updating the account does not change existing Pod environment variables; restart after correcting the identity.

```sh
kubectl -n external-dns get serviceaccount external-dns \
  -o jsonpath='{.metadata.annotations.azure\.workload\.identity/client-id}{"\n"}'
kubectl -n external-dns get pods -l app.kubernetes.io/name=external-dns \
  -o jsonpath='{.items[0].spec.containers[0].env[?(@.name=="AZURE_CLIENT_ID")].value}{"\n"}'
kubectl -n external-dns rollout restart deployment/external-dns
```

## Check public Argo CD access

**Problem:** Browser rejects the HTTPS certificate.

**Solution:** `cert-manager.local` is temporary; Let's Encrypt staging is also untrusted. Fix any HTTP-01 `502` or DNS failure, select `letsencrypt-prod`, and wait for the Certificate to become ready.

```sh
openssl s_client -connect argocd.jkslabs.site:443 -servername argocd.jkslabs.site \
  </dev/null 2>/dev/null | openssl x509 -noout -issuer -dates -ext subjectAltName
kubectl -n argocd get certificate argocd-ingress-tls
kubectl -n argocd get challenges.acme.cert-manager.io \
  -o custom-columns='NAME:.metadata.name,STATE:.status.state,REASON:.status.reason'
kubectl -n argocd wait --for=condition=Ready certificate/argocd-ingress-tls --timeout=300s
```

For workload certificates, run the same checks in the workload namespace.
