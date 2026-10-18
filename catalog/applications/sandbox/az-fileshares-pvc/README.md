# Setup

Loadbalaner
|
Deployment
|
Azure storage account
|
Azure fileshares

Mount the azure fileshares as volumes and provision the fileshares as PVC (Persistent Volume Claim)

Users can access `.html` files from the fileshares to display.
For e.g: http://<lb-external-ip>/app/file1.html

By default, azure auto provisions `Storage Class` in the cluster

```sh
kubectl get sc
```
