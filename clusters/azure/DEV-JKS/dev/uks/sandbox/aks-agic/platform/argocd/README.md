# Argo CD cluster configuration

- `kustomization.yaml`: imports the catalog Application and applies cluster patches.
- `application-patches.yaml`: selects Azure defaults, contains cluster Helm values
  under `helm.valuesObject`, and gates automatic sync.

Terraform's minimal Helm bootstrap does not read this directory. Its root
Application uses `default` and selects the cluster root. This overlay imports
the catalog Application and `bootstrap` project, which are created by the root.

The parent platform Kustomization sets the values Git revision to `dev/sandbox`.
The child merges shared, Azure, then cluster values. It has no Git config source
pointing back at this Application overlay. Keep the automatic-sync gate until
Terraform relinquishes the Helm release and manual sync verifies adoption.

See [bootstrap, handoff and recovery](../../../../../../../../../docs/bootstrap.md).
