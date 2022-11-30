# waofed

[![GitHub](https://img.shields.io/github/license/Nedopro2022/waofed)](https://github.com/Nedopro2022/waofed/blob/main/LICENSE)
[![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/Nedopro2022/waofed)](https://github.com/Nedopro2022/waofed/releases/latest)
[![CI](https://github.com/Nedopro2022/waofed/actions/workflows/ci.yaml/badge.svg)](https://github.com/Nedopro2022/waofed/actions/workflows/ci.yaml)
![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/Nedopro2022/waofed)
[![Go Report Card](https://goreportcard.com/badge/github.com/Nedopro2022/waofed)](https://goreportcard.com/report/github.com/Nedopro2022/waofed)
[![codecov](https://codecov.io/gh/Nedopro2022/waofed/branch/main/graph/badge.svg)](https://codecov.io/gh/Nedopro2022/waofed)

Optimizes workload allocation on KubeFed.

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->

- [Overview](#overview)
- [Getting Started](#getting-started)
  - [Installation](#installation)
  - [Deploy a `WAOFedConfig` resource](#deploy-a-waofedconfig-resource)
    - [Scheduling settings (RSPOptimizer)](#scheduling-settings-rspoptimizer)
  - [Deploy `FederatedDeployment` resources](#deploy-federateddeployment-resources)
  - [Load balancing settings](#load-balancing-settings)
  - [Uninstallation](#uninstallation)
- [Developing](#developing)
  - [Prerequisites](#prerequisites)
  - [Run development clusters with kind](#run-development-clusters-with-kind)
  - [Run tests on kind clusters](#run-tests-on-kind-clusters)
- [License](#license)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

## Overview

WAOFed optimizes workload allocation on [KubeFed](https://github.com/kubernetes-sigs/kubefed) with the following components:

- **RSPOptimizer**: Optimizes `FederatedDeployment` [weights](https://github.com/kubernetes-sigs/kubefed/blob/master/docs/userguide.md#distribute-total-replicas-in-weighted-proportions) across clusters by generating [`ReplicaSchedulingPreference`](https://github.com/kubernetes-sigs/kubefed/blob/master/docs/userguide.md#replicaschedulingpreference) using the specified method.
- // TODO: load balancer

## Getting Started

Supported Kubernetes versions: __1.19 or higher__

> 💡 Mainly tested with 1.25, may work with the same versions that [KubeFed supports](https://github.com/kubernetes-sigs/kubefed/blob/master/docs/userguide.md#create-clusters) (but may require some efforts).

Supported KubeFed APIs:
- `FederatedDeployment [types.kubefed.io/v1beta1]`
- `ReplicaSchedulingPreference [scheduling.kubefed.io/v1alpha1]`
- `KubeFedCluster [core.kubefed.io/v1beta1]`

### Installation

Make sure you have [cert-manager](https://cert-manager.io/) deployed on the cluster where KubeFed control plane is deployed, as it is used to generate webhook certificates.

```sh
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.10.0/cert-manager.yaml
```

> ⚠️ You may have to wait a second for cert-manager to be ready.

Deploy the Operator with the following command. It creates `waofed-system` namespace and deploys CRDs, controllers and other resources.

```sh
kubectl apply -f https://github.com/Nedopro2022/waofed/releases/download/v0.2.1/waofed.yaml
```

### Deploy a `WAOFedConfig` resource

`WAOFedConfig` is a resource for configuring WAOFed. Deploy it with the name `default` to the cluster where KubeFed control plane is deployed.

`spec.kubefedNamespace` specifies the namespace from which WAOFed gets member clusters.

```yaml
apiVersion: waofed.bitmedia.co.jp/v1beta1
kind: WAOFedConfig
metadata:
  name: default # must be default
spec:
  kubefedNamespace: "kube-federation-system"
  scheduling:
    selector:
      hasAnnotation: waofed.bitmedia.co.jp/scheduling
    optimizer:
      method: "rr" # "wao" is not yet implemented
```

#### Scheduling settings (RSPOptimizer)

RSPOptimizer watches the creation of `FederatedDeployment` resources and generates `ReplicaSchedulingPreference` resources with optimized workload allocation determined by the specified method.

Supported methods: `rr` (Round-robin, for testing purposes)

`spec.scheduling.selector` specifies the conditions for the `FederatedDeployment` resources that KubeFed watches.

> 💡 You can enable RSPOptimizer by default by setting `spec.scheduling.selector.any` to true.
>
> ```diff
>    scheduling:
>      selector:
> -      hasAnnotation: waofed.bitmedia.co.jp/scheduling
> +      any: true
> ```

### Deploy `FederatedDeployment` resources

> 💡 Ensure the namespace is federated by a `FederatedNamespace` resource before deploying `FederatedDeployment` resources.
> 
> ```yaml
> apiVersion: types.kubefed.io/v1beta1
> kind: FederatedNamespace
> metadata:
>   name: default
>   namespace: default
> spec:
>   placement:
>     clusterSelector: {}
> ```

When a `FederatedDeployment` with the annotation specified in WAOFedConfig is deployed, RSPOptimizer will detect the resource and generate an `ReplicaSchedulingPreference`.

```yaml
apiVersion: types.kubefed.io/v1beta1
kind: FederatedDeployment
metadata:
  name: fdeploy-sample
  namespace: default
  annotations:
    waofed.bitmedia.co.jp/scheduling: ""
spec:
  template:
    metadata:
      labels:
        app: nginx
    spec:
      replicas: 9
      selector:
        matchLabels:
          app: nginx
      template:
        metadata:
          labels:
            app: nginx
        spec:
          containers:
            - image: nginx:1.23.2
              name: nginx
  placement:
    clusterSelector: {}
```


> 💡 You can see the resources with the following commands, and see the details by adding `-oyaml`.
> ```sh
> $ kubectl get fdeploy
> NAME             AGE
> fdeploy-sample   12s
> 
> $ kubectl get rsp
> NAME             AGE
> fdeploy-sample   12s
> ```

The generated `ReplicaSchedulingPreference` has an owner reference indicating that it is controlled by the `FederatedDeployment` so that it will be deleted by [GC](https://kubernetes.io/docs/concepts/architecture/garbage-collection/) when the `FederatedDeployment` is deleted.

`spec.clusters` includes all clusters specified in `FederatedDeployment` `spec.placement` (RSPOptimizer parses the selector and retrives clusters), and `spec.clusters[name].weight` is optimized by the method specified in `WAOFedConfig`. This sample uses `rr` so all clusters have a weight of 1.

```yaml
apiVersion: scheduling.kubefed.io/v1alpha1
kind: ReplicaSchedulingPreference
metadata:
  name: fdeploy-sample
  namespace: default
  ownerReferences:
  - apiVersion: types.kubefed.io/v1beta1
    blockOwnerDeletion: true
    controller: true
    kind: FederatedDeployment
    name: fdeploy-sample
    ...
spec:
  clusters:
    cluster1:
      weight: 1
    cluster2:
      weight: 1
    cluster3:
      weight: 1
  intersectWithClusterSelector: true
  rebalance: true
  targetKind: FederatedDeployment
  totalReplicas: 9
...
```

> 💡 Since `spec.intersectWithClusterSelector` is set to `true`, the generated `ReplicaSchedulingPreference` does not overwrite anything in the `FederatedDeployment`, allowing RSPOptimizer to watch the `FederatedDeployment` easily.

> ⚠️ **Edge cases not covered:**
>
> **`placement.clusters` has 0 items**
>
> KubeFed ignores `spec.placement.clusterSelector` if `spec.placement.clusters` is provided, so no clusters will be selected for the following case ([docs](https://github.com/kubernetes-sigs/kubefed/blob/master/docs/userguide.md#both-specplacementclusters-and-specplacementclusterselector-are-provided)). However, RSPOptimizer currently does not recognize whether a list is nil (`null`) or has 0 items (`[]`), so it regards `spec.placement.clusters` as "not provided" and uses `spec.placement.clusterSelector` for scheduling.
> ```yaml
> spec:
>   placement:
>     clusters: []
>     clusterSelector:
>       matchExpressions:
>         - { key: mylabel, operator: Exists }
> ```

### Load balancing settings

// TODO: not yet implemented


### Uninstallation

Delete the Operator and resources with the following command.

```sh
kubectl delete -f https://github.com/Nedopro2022/waofed/releases/download/v0.2.1/waofed.yaml
```

## Developing

This Operator uses [Kubebuilder](https://github.com/kubernetes-sigs/kubebuilder), so we basically follow the Kubebuilder way. See the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html) for details.

### Prerequisites

Make sure you have the following tools installed:

- Git
- Make
- Go
- Docker

### Run development clusters with [kind](https://kind.sigs.k8s.io/)

The script creates K8s clusters `kind-waofed-[0123]`, deploys KubeFed control plane on `kind-waofed-0` and let the remaining clusters join as member clusters.

```sh
./hack/dev-kind-reset-clusters.sh
./hack/dev-kind-deploy.sh
```

> ⚠️ NOTE: Currently it is needed to re-create the clusters on every reboot as the script does not set static IPs to Docker containers.

### Run tests on kind clusters

The script creates K8s clusters `kind-waofed-test-[01]`, deploys KubeFed control plane on `kind-waofed-test-0`, let all clusters join as member clusters and runs integration tests.

```sh
./test/rspoptimizer-reset-clusters.sh
./test/rspoptimizer-run-tests.sh
```

## License

Copyright 2022 Bitmedia Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
