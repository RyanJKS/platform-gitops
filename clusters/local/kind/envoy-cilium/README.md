# Setup

## Commands

Create cluster

```sh
kind create cluster --name envoy-cilium --config kind-config.yaml
```

Install Gateway API

```sh
kubectl apply --server-side -f https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.6.1/standard-install.yaml
```

Find the correct host for control plane before installing cilium.

1. Find correct endpoint from current context.

```sh
KUBE_CONTEXT=$(kubectl config current-context)
KIND_CLUSTER=${KUBE_CONTEXT#kind-}
CONTROL_PLANE="${KIND_CLUSTER}-control-plane"

kind get nodes --name "$KIND_CLUSTER"

API_IP=$(docker inspect "$CONTROL_PLANE" \
  --format '{{(index .NetworkSettings.Networks "kind").IPAddress}}')

printf 'Context: %s\nControl plane: %s\nAPI IP: %s\n' \
  "$KUBE_CONTEXT" "$CONTROL_PLANE" "$API_IP"
```

Expect real IP, usually 172.18.x.x. If inspection fails or IP is empty, stop here.

Install Cilium

```sh
helm install cilium cilium/cilium \
  --namespace kube-system \
  --version 1.20.3 \
  -f cilium-values.yaml
```

Debug cilium issues with:

```sh
kubectl -n kube-system logs -l app.kubernetes.io/name=cilium-operator
kubectl -n kube-system logs -l app.kubernetes.io/name=cilium-envoy
kubectl -n kube-system logs -l app.kubernetes.io/name=cilium-agent

```
