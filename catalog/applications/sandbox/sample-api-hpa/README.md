# Sample API with HPA

- Root selects `configs/`: namespace, LimitRange, ResourceQuota, Deployment, Service, and HPA.
- Namespace: `sample-api`; HPA requires Metrics Server.
- Review image, resources, and Service settings in `configs/`.

```sh
kubectl kustomize catalog/applications/sandbox/sample-api-hpa
```
