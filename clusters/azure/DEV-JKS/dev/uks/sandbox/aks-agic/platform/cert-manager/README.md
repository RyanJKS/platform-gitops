**Deploy ExternalDNS through Argo CD**

Use the upstream `external-dns` Helm chart in namespace `external-dns`, with a pinned version and these values:

```yaml
provider:
  name: azure

sources:
  - ingress

domainFilters:
  - jkslabs.site

policy: upsert-only
registry: txt
txtOwnerId: aks-sandbox
txtPrefix: externaldns-

serviceAccount:
  create: true
  name: external-dns
  annotations:
    azure.workload.identity/client-id: "<external-dns-uami-client-id>"

podLabels:
  azure.workload.identity/use: "true"

extraVolumes:
  - name: azure-config
    secret:
      secretName: external-dns-azure

extraVolumeMounts:
  - name: azure-config
    mountPath: /etc/kubernetes
    readOnly: true
```

!!! Note: add this config as another way!

In the same namespace, create the configuration Secret:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: external-dns-azure
  namespace: external-dns
type: Opaque
stringData:
  azure.json: |
    {
      "tenantId": "<tenant-id>",
      "subscriptionId": "<subscription-id>",
      "resourceGroup": "<dns-zone-resource-group>",
      "useWorkloadIdentityExtension": true
    }
```

This contains configuration identifiers, not an identity password. ExternalDNS reads Ingress hostnames and their published gateway address, then maintains DNS records and TXT ownership records. external-dns

The installation should provide:

```yaml
apiVersion: networking.k8s.io/v1
kind: IngressClass
metadata:
  name: azure-application-gateway
spec:
  controller: azure/application-gateway
```

Using the newly created `IngressClass`, deploy the workload's `Ingress`.

**Deploy application Ingresses and request certificates**

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: chaos
  namespace: chaos
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt-staging
    appgw.ingress.kubernetes.io/health-probe-path: /_stcore/health
spec:
  ingressClassName: azure-application-gateway
  tls:
    - hosts:
        - chaos.jkslabs.site
      secretName: chaos-tls
  rules:
    - host: chaos.jkslabs.site
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: chaos-generator
                port:
                  number: 8501
```
