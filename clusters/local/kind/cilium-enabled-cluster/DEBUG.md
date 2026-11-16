# kind / Cilium node readiness

## Issue:

After creating `cilium-lab` with kind and installing Cilium through Helm, the
control-plane node stayed `NotReady`. Three worker containers were running but
had not registered with Kubernetes. Cilium was stuck in `Init:0/6`, kube-proxy
was restarting, and CoreDNS and Hubble pods were pending.

## steps:

Commands below were used during diagnosis. Run them from this directory.
Kubernetes commands explicitly target `kind-cilium-lab`.

1. Read the setup and confirm the Kubernetes context:

   ```sh
   cat README.md kind-config.yaml cilium-values.yaml
   kubectl config current-context
   ```

   `kind-config.yaml` disables the default CNI. The cluster therefore needs
   Cilium to initialize networking before nodes can become Ready.

2. Inspect node conditions and system pods:

   ```sh
   kubectl --context kind-cilium-lab --request-timeout=10s get nodes -o wide
   kubectl --context kind-cilium-lab --request-timeout=10s describe nodes
   kubectl --context kind-cilium-lab --request-timeout=10s \
     -n kube-system get pods -o wide
   ```

   The control plane reported `KubeletNotReady`, `NetworkReady=false`, and
   `NetworkPluginNotReady`. Memory, disk, and PID pressure were false.

3. Check the installed Helm release and its actual values:

   ```sh
   helm --kube-context kind-cilium-lab -n kube-system list
   helm --kube-context kind-cilium-lab -n kube-system get values cilium
   ```

   Cilium `1.20.3` was installed with Kubernetes IP address management and
   Hubble relay/UI enabled. The local `cilium-values.yaml` had not been applied.

4. Read the failing Cilium init container and kube-proxy logs:

   ```sh
   kubectl --context kind-cilium-lab --request-timeout=10s \
     -n kube-system logs -l k8s-app=cilium -c config --tail=100
   kubectl --context kind-cilium-lab --request-timeout=10s \
     -n kube-system logs -l k8s-app=cilium -c config --previous --tail=12
   kubectl --context kind-cilium-lab --request-timeout=10s \
     -n kube-system logs -l k8s-app=kube-proxy --tail=80
   kubectl --context kind-cilium-lab --request-timeout=10s \
     -n kube-system logs -l io.cilium/app=operator --tail=80
   ```

   Cilium's `config` init container could not reach the API through the
   Kubernetes service IP:

   ```text
   Unable to contact k8s api-server
   dial tcp 10.96.0.1:443: i/o timeout
   ```

   kube-proxy reported the underlying failure:

   ```text
   failed complete: fsnotify watcher init: too many open files
   ```

The system had run out of “file watchers”—slots programs use to notice when files change.

That stopped Kubernetes’ network routing component, kube-proxy, from starting. Cilium then couldn’t reach Kubernetes to finish setting up networking, so the node stayed NotReady.

5. Inspect Docker containers, kernel limits, and worker kubelet logs:

   ```sh
   docker ps --format '{{.Names}} {{.Image}}'
   docker exec cilium-lab-control-plane \
     sysctl fs.inotify.max_user_instances fs.inotify.max_user_watches fs.file-max
   docker exec cilium-lab-control-plane sh -c 'ulimit -n; ls /etc/cni/net.d; ls /kind'
   docker exec cilium-lab-worker journalctl -u kubelet -n 12 --no-pager
   ```

   The worker journal was checked during recovery. Observed limits were:

   | Setting                         | Value                 |
   | ------------------------------- | --------------------- |
   | `fs.inotify.max_user_instances` | `128`                 |
   | `fs.inotify.max_user_watches`   | `524288`              |
   | `fs.file-max`                   | `9223372036854775807` |
   | Open file limit (`ulimit -n`)   | `1048576`             |

## RCA:

The Docker host's inotify instance limit of `128` was exhausted. kind containers share the host kernel, so several nodes and clusters can compete for that limit.
This was an inotify instance shortage despite the high open-file limit.

kube-proxy could not create its filesystem watcher and crashed. Without kube-proxy's service routing, Cilium's init container could not connect to `10.96.0.1:443`, the Kubernetes API service address. Cilium could not finish initializing the CNI, leaving the node `NotReady`.

Raising only `fs.inotify.max_user_instances` restored kube-proxy, allowed the
workers to register, and let Cilium initialize successfully.

## Solution:

These recovery commands were applied:

```sh
docker exec cilium-lab-control-plane \
  sysctl -w fs.inotify.max_user_instances=1024
kubectl --context kind-cilium-lab --request-timeout=10s \
  -n kube-system rollout restart daemonset/kube-proxy
kubectl --context kind-cilium-lab --request-timeout=10s \
  -n kube-system rollout status daemonset/kube-proxy --timeout=45s
```

The sysctl command changes the shared Docker host kernel setting at runtime.
Cilium and kubelet retried automatically; no cluster recreation or Cilium
reinstallation was needed.

Verify recovery:

```sh
kubectl --context kind-cilium-lab --request-timeout=10s get nodes -o wide
kubectl --context kind-cilium-lab --request-timeout=10s \
  -n kube-system get pods
kubectl --context kind-cilium-lab --request-timeout=10s \
  -n kube-system logs -l k8s-app=cilium -c config --tail=8
```

Verified result: all four nodes became `Ready`; all Cilium agents, Envoy pods,
operators, kube-proxy pods, CoreDNS, Hubble relay, and Hubble UI were ready.
Cilium's init logs reported `Connected to apiserver`. Application connectivity
tests were not run.

The runtime sysctl change does not persist across host restarts. For persistence,
configure `fs.inotify.max_user_instances=1024` under `/etc/sysctl.d/` on the Linux
host running the Docker daemon and ensure that host applies it at boot. With
Docker Desktop, that host may be a separate VM from the WSL distribution. No
persistent host configuration was changed during this recovery.
