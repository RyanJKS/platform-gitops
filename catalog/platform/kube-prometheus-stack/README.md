https://github.com/prometheus-community/helm-charts

```sh
helm install prometheus prometheus-community/kube-prometheus-stack --version 92.7.0 --namespace observability --create-namespace
```

if installation succeded, the following should appear

```sh
NAME: prometheus
LAST DEPLOYED: Wed Oct 14 23:28:59 2026
NAMESPACE: observability
STATUS: deployed
REVISION: 1
DESCRIPTION: Install complete
NOTES:
kube-prometheus-stack has been installed. Check its status by running:
  kubectl --namespace observability get pods -l "release=prometheus"

Get Grafana 'admin' user password by running:

  kubectl --namespace observability get secrets prometheus-grafana -o jsonpath="{.data.admin-password}" | base64 -d ; echo

Access Grafana local instance:

  export POD_NAME=$(kubectl --namespace observability get pod -l "app.kubernetes.io/name=grafana,app.kubernetes.io/instance=prometheus" -oname)
  kubectl --namespace observability port-forward $POD_NAME 3000

Get your grafana admin user password by running:

  kubectl get secret --namespace observability -l app.kubernetes.io/component=admin-secret -o jsonpath="{.items[0].data.admin-password}" | base64 --decode ; echo


Visit https://github.com/prometheus-operator/kube-prometheus for instructions on how to create & configure Alertmanager and Prometheus instances using the Operator.
```

Look into prometheus data

```sh
kubectl port-forward -n observability svc/prometheus-operated 9090:9090
```

Grafana page using prometheus data

```sh

kubectl port-forward -n observability svc/prometheus-grafana 3000:80
```

Get Grafana secret.. username is "admin"

```sh
kubectl get secret -n observability prometheus-grafana -o jsonpath="{.data.admin-password}" | base64 --decode

```
