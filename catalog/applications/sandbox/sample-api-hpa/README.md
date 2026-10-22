# Setup

Test load to app with the following:

```sh
kubectl run apache-bench --i --tty --rm --image=httpd -- ab -n 50000 -c 1000 http:<service-name>.<namespace-name>.svc.cluster.local/
```
