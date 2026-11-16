Pool

| Pool            | Example VM             | Min–max nodes               | Workloads                                    | What you learn                                     |
| --------------- | ---------------------- | --------------------------- | -------------------------------------------- | -------------------------------------------------- |
| Existing system | Keep regular pricing   | Your existing configuration | CoreDNS, metrics-server, managed components  | System/user separation                             |
| `general`       | `Standard_D2_v5`       | **0–3**                     | APIs, web apps, CPU exercises, Spark drivers | Selectors, affinity, anti-affinity, spreading      |
| `memory`        | `Standard_E2_v5`       | **0–2**                     | Memory-heavy processing, Spark executors     | Memory requests, limits, OOM, dedicated scheduling |
| `spot`          | `Standard_D2_v5`, Spot | **0–2**                     | Disposable batch jobs                        | Multiple tolerations, interruption handling        |
| `gpu`           | `Standard_NC4as_T4_v3` | **0–1**                     | GPU tests, training or inference later       | Device plugins, extended resources                 |

The D2 example has 2 vCPUs and 8 GiB RAM; E2 has 2 vCPUs and 16 GiB RAM. These are illustrative sizes, subject to availability and quota in your region. Microsoft Learn

User pools can autoscale to zero. **`min_count = 0` permits scale-down; it doesn’t force zero while workloads still need the nodes.** Microsoft Learn

The archive’s `terragrunt-inputs.hcl` contains the complete map and shared defaults for the attributes accessed by your module. Merge it into your existing `locals` and `inputs` blocks.

Each pool uses this arrangement:

| Pool    | Node label                       | Node taint                                             |
| ------- | -------------------------------- | ------------------------------------------------------ |
| General | `lab.jkslabs.site/class=general` | `workload=general:NoSchedule`                          |
| Memory  | `lab.jkslabs.site/class=memory`  | `workload=memory:NoSchedule`                           |
| Spot    | `lab.jkslabs.site/class=batch`   | `workload=batch:NoSchedule` **and** Azure’s Spot taint |
| GPU     | `lab.jkslabs.site/class=gpu`     | `workload=gpu:NoSchedule`                              |

This gives you a useful pairing: **the label lets a workload select the pool; the taint keeps unrelated workloads out.**

**1. General applications: selector plus toleration**

This is the basic pattern. It belongs inside a Deployment’s `spec.template.spec`:

```
spec:
  template:
    spec:
      nodeSelector:
        lab.jkslabs.site/class: general

      tolerations:
        - key: workload
          operator: Equal
          value: general
          effect: NoSchedule

      containers:
        - name: web
          image: nginx:1.28-alpine
          resources:
            requests:
              cpu: 100m
              memory: 64Mi
            limits:
              cpu: 500m
              memory: 128Mi
```

The selector **requires** general nodes. The toleration **allows** their taint. A toleration by itself doesn’t make a pod choose that pool. Kubernetes

Use `manifests/01-general.yaml` for the complete Deployment.

**2. Memory-heavy workloads: required node affinity**

Instead of an exact `nodeSelector`, practice the more expressive node-affinity syntax:

```
spec:
  template:
    spec:
      tolerations:
        - key: workload
          operator: Equal
          value: memory
          effect: NoSchedule

      affinity:
        nodeAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
            nodeSelectorTerms:
              - matchExpressions:
                  - key: lab.jkslabs.site/class
                    operator: In
                    values:
                      - memory

      containers:
        - name: processor
          image: python:3.12-slim
          command:
            - python
            - -u
            - -c
            - |
              import time
              data = bytearray(512 * 1024 * 1024)
              for offset in range(0, len(data), 4096):
                  data[offset] = 1
              print("Allocated 512 MiB", flush=True)
              time.sleep(180)
          resources:
            requests:
              cpu: 500m
              memory: 1Gi
            limits:
              cpu: "1"
              memory: 2Gi
```

The archive wraps this as a finite Job in `05-memory-job.yaml`.

Useful experiments:

- Request **100Gi**: observe Pending because none of your proposed nodes can fit it.
- Set the limit to **256Mi** while retaining the 512 MiB allocation: observe container `OOMKilled`.
- Change required affinity to preferred: observe that placement becomes a preference.

**3. Spot batch workloads: tolerate both taints**

Your Spot pool has two taints, so its workload needs both tolerations:

```
spec:
  template:
    spec:
      restartPolicy: Never

      nodeSelector:
        lab.jkslabs.site/class: batch

      tolerations:
        - key: workload
          operator: Equal
          value: batch
          effect: NoSchedule

        - key: kubernetes.azure.com/scalesetpriority
          operator: Equal
          value: spot
          effect: NoSchedule

      containers:
        - name: batch
          image: busybox:1.37
          command:
            - sh
            - -c
            - "echo Starting batch; sleep 120; echo Finished"
          resources:
            requests:
              cpu: 100m
              memory: 32Mi
            limits:
              cpu: 500m
              memory: 64Mi
```

Use `06-spot-job.yaml`. Remove either toleration and inspect the scheduling events. Azure applies the Spot label and taint to Spot nodes. learn.microsoft.com

**4. GPU workloads: request an actual GPU**

A GPU label controls placement. **`nvidia.com/gpu` requests the hardware resource.**

```
apiVersion: batch/v1
kind: Job
metadata:
  name: gpu-demo
  namespace: scheduling-lab
spec:
  backoffLimit: 1
  activeDeadlineSeconds: 1200
  ttlSecondsAfterFinished: 600
  template:
    spec:
      restartPolicy: Never

      nodeSelector:
        lab.jkslabs.site/class: gpu

      tolerations:
        - key: workload
          operator: Equal
          value: gpu
          effect: NoSchedule

      containers:
        - name: gpu-check
          image: nvidia/cuda:12.4.1-base-ubuntu22.04
          command: ["nvidia-smi"]
          resources:
            requests:
              cpu: 500m
              memory: 256Mi
            limits:
              cpu: "1"
              memory: 1Gi
              nvidia.com/gpu: 1
```

Your shown resource doesn’t enable AKS-managed GPU mode. With the default driver setup on supported GPU node images, you must also install the **NVIDIA device plugin**. The archive includes a plugin DaemonSet that tolerates your custom GPU taint. Microsoft Learn

```
kubectl apply -f gpu/00-device-plugin.yaml
kubectl apply -f gpu/01-gpu-job.yaml

kubectl -n scheduling-lab logs job/gpu-demo
kubectl describe node <gpu-node>
```

Check that the node advertises `nvidia.com/gpu: 1`. This first example tests GPU allocation and visibility; model training can follow.

**5. Spark: separate driver and executor placement**

**Spark doesn’t require its own pool.** For your memory-heavy experiment, use:

| Spark component           | Pool                       | Reason                               |
| ------------------------- | -------------------------- | ------------------------------------ |
| Driver                    | `general`, regular pricing | Coordinates the application          |
| Executors                 | `memory`                   | Run the processing tasks             |
| Later executor experiment | Memory-sized Spot pool     | Test executor loss and recomputation |

Spark supports separate selectors and pod templates for drivers and executors. Put the tolerations in their corresponding templates. Spark 4.2.0 Documentation

```
spark.kubernetes.driver.node.selector.lab.jkslabs.site/class general
spark.kubernetes.executor.node.selector.lab.jkslabs.site/class memory

spark.kubernetes.driver.podTemplateFile /opt/spark/conf/driver-template.yaml
spark.kubernetes.executor.podTemplateFile /opt/spark/conf/executor-template.yaml

spark.driver.memory 1g
spark.executor.instances 2
spark.executor.cores 1
spark.executor.memory 4g
spark.executor.memoryOverhead 1g
```

The archive’s Spark files are **scheduling inputs for a Spark-on-Kubernetes application**, rather than a complete Spark installation. You still supply the image, application, submission method, and service-account permissions.

**Use the general pool for these placement exercises:**

| Exercise                | Configuration                                                | Expected result                                             |
| ----------------------- | ------------------------------------------------------------ | ----------------------------------------------------------- |
| Pod affinity            | Require proximity to pods labelled `app=cache`               | Client runs on the same node as a cache pod                 |
| Pod anti-affinity       | Require separation from pods with the same application label | Three replicas require three different nodes                |
| Preferred anti-affinity | Use a preference instead of a requirement                    | Replicas may share nodes                                    |
| Topology spread         | Balance replicas across hostnames                            | Three replicas can distribute 2+1 across two eligible nodes |
| Zone spreading          | Spread using `topology.kubernetes.io/zone`                   | Requires actual nodes in multiple zones                     |

Here is **hard pod anti-affinity**, inside `spec.template.spec`, for pods labelled `app=anti-web`:

```
affinity:
  podAntiAffinity:
    requiredDuringSchedulingIgnoredDuringExecution:
      - labelSelector:
          matchLabels:
            app: anti-web
        topologyKey: kubernetes.io/hostname
```

Three replicas then need three eligible nodes. If the pool maximum is two, one replica remains Pending. Pod affinity/anti-affinity examines **other pods’ labels**, whereas node affinity examines **nodes’ labels**. Kubernetes

Compare it with topology spread:

```
topologySpreadConstraints:
  - maxSkew: 1
    topologyKey: kubernetes.io/hostname
    whenUnsatisfiable: DoNotSchedule
    nodeAffinityPolicy: Honor
    nodeTaintsPolicy: Honor
    labelSelector:
      matchLabels:
        app: spread-web
```

This balances counts across eligible domains. **It doesn’t demand one pod per node:** if only one eligible node exists, all three pods may fit there. Kubernetes

**Run one exercise at a time:**

```
kubectl apply -f manifests/00-namespace.yaml
kubectl apply -f manifests/01-general.yaml

kubectl -n scheduling-lab get pods -o wide --watch
kubectl -n scheduling-lab describe pod <pod-name>
kubectl -n scheduling-lab get events --sort-by=.metadata.creationTimestamp

# Finish this exercise before starting the next
kubectl delete -f manifests/01-general.yaml
kubectl apply -f manifests/02-anti-affinity.yaml
```

Then progress to resource requests/limits, HPA versus node autoscaling, PriorityClass, PodDisruptionBudget, StatefulSet/PVC, and namespace quotas. Those lessons can use the same pools.

**For cost control, create GPU last and remove active exercises when finished.** A Deployment with running replicas keeps its required capacity in use. If Argo CD manages the examples, remove or disable them in Git as well so they aren’t recreated.

The downloadable YAML passed local structural checks. It hasn’t been applied to a cluster; review `terragrunt run plan` and use server-side dry runs against your cluster before deploying.
