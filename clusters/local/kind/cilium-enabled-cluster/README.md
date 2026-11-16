# Setup

Run these commands from this directory. On Linux or WSL, kind nodes share
the Docker host kernel's inotify limits. If several nodes or clusters exhaust
the default limit, kubelet or kube-proxy can fail with
`fsnotify watcher init: too many open files`.

Check the limit on the Linux host running Docker:

```sh
sysctl fs.inotify.max_user_instances
```

If the limit is exhausted, raise it before creating the cluster:

```sh
sudo sysctl -w fs.inotify.max_user_instances=1024
```

This is a host-wide runtime setting. To persist it across restarts, configure
`fs.inotify.max_user_instances=1024` in the Docker host's `/etc/sysctl.d/`
configuration and ensure that host applies it at boot. With Docker Desktop,
the Docker host may be a separate VM from your WSL distribution.

Install cluster

```sh

kind create cluster \
  --name cilium-lab \
  --config kind-config.yaml

kubectl config use-context kind-cilium-lab
```

Install cilium

```sh

helm repo add cilium https://helm.cilium.io/
helm repo update

helm install cilium cilium/cilium \
  --version 1.20.3 \
  --namespace kube-system \
  --set ipam.mode=kubernetes \
  --set hubble.relay.enabled=true \
  --set hubble.ui.enabled=true
```

or

```sh
helm install cilium cilium/cilium \
  --namespace kube-system \
  --version <CILIUM_VERSION> \
  -f cilium-values.yaml
```

The values-file alternative also enables Prometheus `ServiceMonitor` resources;
install the Prometheus Operator CRDs before using it. The inline command above
does not require those CRDs.

Validate

```sh

cilium status --wait
cilium connectivity test

kubectl get nodes
kubectl get pods -n kube-system
```

## Troubleshooting NotReady nodes

See [DEBUG.md](DEBUG.md) for the diagnostic commands, root cause, recovery,
and verified result for the inotify exhaustion issue.

## Access Hubble dashboard

```

cilium status

cilium hubble port-forward &
hubble observe --follow
```

And open the UI through a separate port-forward:

```

kubectl -n kube-system port-forward \
  svc/hubble-ui 12000:80
```

Then visit `http://localhost:12000`.

You can inspect live flows using the Hubble CLI:

```

# Observe dropped packets
hubble observe --verdict DROPPED

# Observe traffic for the demo namespace
hubble observe --namespace demo

# Observe traffic to the backend
hubble observe --to-pod demo/backend

# Follow traffic continuously
hubble observe --follow
```

```

```
