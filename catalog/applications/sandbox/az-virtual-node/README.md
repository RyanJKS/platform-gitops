# Setup

Requires infra setup correctly first

**You don’t create an Azure Container Instance resource first.** `aci_connector_linux` enables the AKS **virtual nodes add-on**, which provisions ACI workloads dynamically when you schedule pods onto its virtual node.

In Terraform, the block points to a **subnet**, rather than an existing container instance:

```
# Inside azurerm_kubernetes_cluster
aci_connector_linux {
  subnet_name = azurerm_subnet.aci.name
}
```

That subnet must be delegated to `Microsoft.ContainerInstance/containerGroups`. HashiCorp recommends declaring the delegation in Terraform because AKS adds it automatically otherwise, which can affect subsequent Terraform runs.

The process works like this:

1. You enable the add-on.
2. AKS exposes a virtual node named `virtual-node-aci-linux`, using **Virtual Kubelet**.
3. You deploy a Kubernetes workload with the node selector and tolerations needed to target that node.
4. The connector creates the backing ACI container group and manages its lifecycle for that pod. Microsoft Learn

**You still control the workload; Azure manages the underlying compute:**

| You control                                        | AKS / ACI manages                             |
| -------------------------------------------------- | --------------------------------------------- |
| Container image, command and environment variables | Provisioning the backing container group      |
| Supported CPU and memory requests                  | Underlying servers                            |
| Deployment replicas and pod scheduling             | Translating scheduled pods into ACI workloads |
| Subnet, delegation and network permissions         | The virtual-node integration                  |

Your workload configuration is expressed through Kubernetes manifests, within the add-on’s supported feature set. Microsoft Learn

For networking, configure a separate delegated ACI subnet and grant the **virtual-node add-on identity** `Network Contributor` on that subnet. The documented add-on requires Azure CNI.

One current distinction: Microsoft now documents **Virtual Nodes v2 (preview)** as a separate cluster extension. The `aci_connector_linux` block refers to the existing virtual nodes add-on discussed above.

Virtual Nodes v2 enablement can be found here using aks_extension terraform resource: https://learn.microsoft.com/en-us/azure/aks/virtual-nodes-v2
