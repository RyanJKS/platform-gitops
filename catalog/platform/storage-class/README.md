# Storage classes

| Selection | Driver | Defaults |
| --- | --- | --- |
| `azure/` | Azure Blob CSI, `blob.csi.azure.com` | `blob-fuse`, `Standard_GRS`, reclaim `Delete`, binding `Immediate` |

- Install Azure Blob CSI driver before selecting this class.
- Review SKU, reclaim policy, and mount options in `azure/blobfuse-sc.yaml`.
- StorageClass is cluster scoped; no namespace configuration needed.

```sh
kubectl kustomize catalog/platform/storage-class/azure
```
