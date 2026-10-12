# AKS UK South AGIC sandbox

The cluster root selects `applications-project.yaml`, `platform-project.yaml`,
`platform/`, and `workloads/`. It renders three AppProjects and five Applications in
namespace `argocd`. Parent Kustomizations select local app folders. Each app's
Kustomization imports its catalog Application and applies local Application
patches. Cluster patches select provider values, child config paths, and Git
revisions; catalog manifests contain no cluster paths. The root does not render controller or workload resources.

Cert-manager, External DNS, and chaos-generator child Git sources deploy separate
`configs/` overlays. Each resource Kustomization and its patches live together;
Application patches stay beside the app-level Kustomization.
Helm children read their selected catalog and
inline cluster `helm.valuesObject` settings in `application-patches.yaml`. Argo CD uses the GitOps-owned `bootstrap` project; the Terraform root uses
`default`; runtime values and the handoff gate live in
[Argo CD configuration](platform/argocd/README.md).

Projects use wave `-4`; AGIC uses `-3`, External DNS `-2`, cert-manager `-1`,
chaos-generator `0`, and Argo CD `1`. Runtime Application health customization
makes the root wait for synced, healthy children. Minimal bootstrap lacks that
customization; verify initial children manually. Existing children reconcile
independently.

Selected Git sources use `dev/sandbox`. Publish reviewed configuration before
reconciliation. Verify Azure identities, gateway access, DNS, and certificates
separately from Application health.

See [bootstrap](../../../../../../../docs/bootstrap.md) for installation,
ownership transfer, and recovery.

**The diagrams separate configuration, identity and live traffic so each relationship stays readable.**

Solid arrows in the first diagram show deployment or resource relationships; dotted arrows show controller reconciliation.

```mermaid
flowchart TB
    TF["Terraform"]
    GIT["GitOps repository"]

    subgraph AZ["Azure"]
        DNS["Azure DNS zone<br/>jkslabs.site"]
        ARM["Azure Resource Manager API"]

        subgraph VNET["Virtual network"]
            subgraph GWNET["Dedicated Application Gateway subnet"]
                GW["Application Gateway<br/>Public frontend IP<br/>Listeners, rules, probes and TLS"]
            end

            subgraph AKS["AKS cluster"]
                API["Kubernetes API"]

                subgraph GLOBAL["Cluster-scoped Kubernetes objects"]
                    IC["IngressClass<br/>azure-application-gateway"]
                    ISS["ClusterIssuer and cert-manager CRDs"]
                    RB["ClusterRoles and ClusterRoleBindings"]
                end

                subgraph ARGO["Namespace: argocd"]
                    AC["Argo CD application-controller<br/>Root and child Applications"]
                end

                subgraph AGICNS["Namespace: ingress-agic"]
                    AG["AGIC Deployment<br/>ServiceAccount: ingress-agic"]
                end

                subgraph DNSNS["Namespace: external-dns"]
                    ED["ExternalDNS Deployment<br/>ServiceAccount: external-dns<br/>Azure configuration"]
                end

                subgraph CERTNS["Namespace: cert-manager"]
                    CM["Controller, webhook and cainjector"]
                end

                WORK["Application namespaces<br/>argocd and chaos<br/>Deployments, Services, Ingresses<br/>Certificates and TLS Secrets"]
            end
        end
    end

    TF -->|"Provision infrastructure"| GW
    TF -->|"Provision zone"| DNS
    TF -->|"Bootstrap Argo CD"| AC

    GIT -->|"Desired manifests and Helm values"| AC
    AC -->|"Apply resources"| API
    API --> IC
    API --> ISS
    API --> RB
    API --> AG
    API --> ED
    API --> CM
    API --> WORK

    AG -.->|"Watch Ingresses, Services, endpoints and TLS Secrets"| API
    ED -.->|"Watch Ingress hosts and status addresses"| API
    CM -.->|"Manage certificates and ACME resources"| API

    AG -.->|"Authenticated gateway updates"| ARM
    ARM -.->|"Apply gateway configuration"| GW
    ED -.->|"Create A and TXT records"| DNS
```

The next diagram shows **authentication and Azure permission scopes**. Federated credentials connect particular ServiceAccounts to particular UAMIs; Kubernetes RBAC is a separate check.

```mermaid
flowchart TB
    subgraph AZ["Azure and Microsoft Entra"]
        CP["Control-plane UAMI"]
        KL["Kubelet UAMI"]
        NET["AKS subnet"]
        ACR["ACR, if used"]

        ENTRA["Microsoft Entra token exchange"]
        AF["AGIC UAMI<br/>Federated credential"]
        DF["ExternalDNS UAMI<br/>Federated credential"]

        GW["Application Gateway<br/>Gateway resource group<br/>Referenced subnet and public IP"]
        DNS["Azure DNS zone<br/>DNS resource group"]

        subgraph AKS["AKS cluster"]
            OIDC["AKS OIDC issuer"]

            subgraph AGNS["Namespace: agic"]
                AGSA["ServiceAccount: ingress-azure<br/>AGIC Pod"]
            end

            subgraph EDNS["Namespace: external-dns"]
                EDSA["ServiceAccount: external-dns<br/>ExternalDNS Pod"]
            end
        end
    end

    CP -->|"Network Contributor"| NET
    CP -->|"Managed Identity Operator"| KL
    KL -->|"Image-pull permission"| ACR

    OIDC -.->|"Issuer used to verify projected tokens"| ENTRA
    AGSA -->|"ServiceAccount token exchange"| ENTRA
    EDSA -->|"ServiceAccount token exchange"| ENTRA

    ENTRA -->|"Matching AGIC federation"| AF
    ENTRA -->|"Matching ExternalDNS federation"| DF

    AF -->|"Gateway Contributor, RG Reader<br/>and network read/join permissions"| GW
    DF -->|"Zone Contributor and RG Reader"| DNS
```

Finally, this is the **browser traffic and certificate-validation path**. Services select the application backends; they are not shown as an additional ClusterIP packet hop.

```mermaid
flowchart TB
    B["Browser"]
    LE["Let's Encrypt"]

    subgraph AZ["Azure"]
        DNS["Azure DNS<br/>Both hostnames resolve to gateway IP"]

        subgraph VNET["Virtual network"]
            subgraph GWNET["Application Gateway subnet"]
                GW["Application Gateway<br/>HTTPS listener: 443<br/>HTTP challenge listener: 80<br/>Host and path routing"]
            end

            subgraph AKS["AKS cluster"]
                subgraph ARGO["Namespace: argocd"]
                    AS["Argo CD ClusterIP Service"]
                    AP["Argo CD server Pods"]
                    AT["argocd-server-tls Secret"]
                end

                subgraph CHAOS["Namespace: chaos"]
                    CS["Chaos ClusterIP Service"]
                    CP["Chaos Pods<br/>Application port: 8501"]
                    CT["chaos-tls Secret"]
                end

                subgraph SOLVER["Temporary resources in requesting application namespace"]
                    SI["Solver Ingress and Service"]
                    SP["ACME solver Pod"]
                end
            end
        end
    end

    B -.->|"DNS lookup"| DNS
    DNS -.->|"Returns public IP"| B

    B -->|"HTTPS to returned IP"| GW
    GW -->|"argocd hostname: backend HTTP"| AP
    GW -->|"chaos hostname: backend HTTP"| CP

    AS -.->|"Selects"| AP
    CS -.->|"Selects"| CP

    AT -.->|"AGIC installs certificate"| GW
    CT -.->|"AGIC installs certificate"| GW

    LE -->|"HTTP domain challenge on port 80"| GW
    GW -->|"ACME challenge path"| SP
    SI -.->|"AGIC configures temporary route"| GW
```

Cert-manager requests and renews certificates; ExternalDNS maintains hostname records; AGIC configures Application Gateway. **Application Gateway is the component that actually receives your users’ HTTP/HTTPS requests.**
