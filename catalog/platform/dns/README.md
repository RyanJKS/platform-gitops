# AKS DNS templates

Copy into `clusters/<...>/platform/dns/`. Run commands from copied directory.
Templates only; cluster must select resources explicitly.

## Why each exists

- **CoreDNS:** cluster DNS pods. Resolve Kubernetes Services; forward external queries upstream. Required for cluster service discovery.
- **LocalDNS:** optional AKS cache on each node. Reduce DNS latency, CoreDNS load, and connection tracking pressure. Still needs CoreDNS.
- **External DNS:** separate controller. Publishes records in cloud DNS; does not resolve pod queries.

Use CoreDNS logging to inspect queries reaching cluster DNS.
Use LocalDNS for caching and node-level troubleshooting. Cache hits never reach CoreDNS logs.

## CoreDNS: enable query logs

[`coredns-custom.yaml`](coredns-custom.yaml) adds `data.log.override: |` containing `log`.
AKS imports `.override` keys into managed CoreDNS configuration.
Merge with existing `coredns-custom` data first; preserve other overrides. Leave managed `coredns` ConfigMap alone.

Requires correct Kubernetes context and permission to update `kube-system` resources.

```sh
kubectl apply -f coredns-custom.yaml
kubectl -n kube-system rollout restart deployment/coredns
kubectl -n kube-system rollout status deployment/coredns
kubectl -n kube-system logs -l k8s-app=kube-dns --prefix --tail=100 -f
```

Run queries from test pod below. Expect query name, response code, and latency in logs.
To stop query logs, remove `log.override` from desired ConfigMap, apply, then repeat restart.
If GitOps manages ConfigMap, commit changes there too.

### Optional internal DNS examples

Same YAML includes commented examples. Replace sample addresses and names, then uncomment selected keys under `data`.

- `internal-hosts.override`: static host records. `ttl 30` sets 30-second TTL; `no_reverse` disables automatic PTR records.
  Keep `fallthrough` so unmatched names still resolve. Merge into existing `hosts` plugin if present; only one per server block.
- `internal-custom.override`: regex alias for names ending `.svc.cluster.local`.
  Example: query `api.apps.aks.com` resolves `api.apps.svc.cluster.local`; response names use `api.apps.aks.com`.
- `internal-default.override`: regex alias for names ending `.default.svc.cluster.local`.
  Example: query `api.aks.com` resolves `api.default.svc.cluster.local`; response names use `api.aks.com`.
- `service-alias.override`: rewrite `api.internal.example` to existing Service `api` in namespace `apps`.
  Service must exist. DNS alias does not configure Gateway routing or TLS certificates.
- `corporate.server`: forward `corp.example` queries to private resolver. Includes separate error and query logging.
  Resolver must be reachable over UDP/TCP 53. Avoid forwarding loops back to CoreDNS.

`.override` adds plugins inside managed root server block. `.server` adds complete server block for separate zone.
`192.0.2.0/24` addresses are documentation examples; replace before enabling.
Apply ConfigMap and repeat CoreDNS restart above after edits.

Choose **one regex option** for `.aks.com`; both match same query suffix.
`name regex` converts alias query to Kubernetes name. `answer name` converts response name back.
`rewrite stop` stops further rewrite rules after matching. Anchors and escaped dots restrict matches to intended suffix.
Trailing dots match fully qualified DNS names on wire.
These aliases preserve actual cluster domain `cluster.local`; they do not publish public DNS records.
`.aks.com` is requested example suffix. Replace with domain you control before deployment.

For regex option A, run `nslookup kubernetes.default.aks.com` from test pod.
For option B, run `nslookup kubernetes.aks.com`. Expect same IP as `kubernetes.default.svc.cluster.local`.

From test pod, run `nslookup registry.internal.example`, `nslookup api.internal.example`, or `nslookup host.corp.example`.
Expect configured IP, Service IP, or private resolver's answer respectively, after replacing sample names.
LocalDNS cache can delay changed answers. `Default` pods using VNet DNS can bypass these CoreDNS overrides.

## LocalDNS: configure node pool

[`localdns.json`](localdns.json) targets AKS managed node pools; deploy through Azure CLI.
Requires Kubernetes 1.31+, Azure CLI 2.80.0+, Azure Linux or Ubuntu 22.04+, and at least 4 vCPUs per node.
Confirm AKS compatibility, especially BYOCNI/Cilium policies. Allow DNS traffic over UDP and TCP port 53.
Do not combine with upstream NodeLocal DNSCache.

**Changing LocalDNS profile reimages nodes, including disabling it. Plan replicas, disruption budgets, and maintenance window.**

Set each shell variable for target node pool. Azure identity needs node pool update permission.

```sh
RESOURCE_GROUP=REPLACE_PLACEHOLDER
AKS_CLUSTER=REPLACE_PLACEHOLDER
NODE_POOL=REPLACE_PLACEHOLDER

az aks nodepool update \
  --resource-group "$RESOURCE_GROUP" \
  --cluster-name "$AKS_CLUSTER" \
  --name "$NODE_POOL" \
  --localdns-config ./localdns.json

az aks nodepool show \
  --resource-group "$RESOURCE_GROUP" \
  --cluster-name "$AKS_CLUSTER" \
  --name "$NODE_POOL" \
  --output json
```

Expect `Succeeded` and configured profile. Repeat per selected pool.

- `Required`: enable or fail validation. `Disabled`: disable using same update command.
- `Preferred`: validation only on 1.31–1.36; can enable on compatible pools from 1.37. Not dry-run.
- `kubeDNSOverrides`: `ClusterFirst` pods. `vnetDNSOverrides`: `Default` pods.
- `cluster.local`: forward to CoreDNS. Root `.`: CoreDNS for `ClusterFirst`, VNet DNS for `Default`.
- `Log`: query logging enabled. Use `Error` after troubleshooting; profile update reimages nodes.
- Cache TTL capped at 3600 seconds. `Immediate` permits stale answers up to 3600 seconds; tune for workload.

On target node, use approved node access and run:

```sh
sudo journalctl -u localdns -f
```

## Verify DNS

Set [`dns-check.yaml`](dns-check.yaml) node selector to target pool. Requires permission to create and exec pods.

```sh
kubectl apply -f dns-check.yaml
kubectl -n default wait --for=condition=Ready pod/dns-check --timeout=120s
kubectl -n default exec dns-check -- nslookup kubernetes.default.svc.cluster.local
kubectl -n default exec dns-check -- nslookup microsoft.com
kubectl -n default delete -f dns-check.yaml
```

Expect successful answers. With LocalDNS active, resolver address should be `169.254.10.10` or `169.254.10.11`.
CoreDNS-only pods use cluster DNS Service IP. Test `Default` policy separately if workloads use it.
If rollout breaks resolution, set JSON `mode` to `Disabled`, then repeat Azure update; expect another reimage.

References: [CoreDNS rewrite](https://coredns.io/plugins/rewrite/),
[CoreDNS logs](https://learn.microsoft.com/en-us/azure/aks/coredns-troubleshoot),
[LocalDNS configuration](https://learn.microsoft.com/en-us/azure/aks/localdns-custom),
[LocalDNS troubleshooting](https://learn.microsoft.com/en-us/troubleshoot/azure/azure-kubernetes/connectivity/dns/troubleshoot-localdns).
