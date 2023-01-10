#!/usr/bin/env bash

# scripts must be run from project root
. hack/2-lib.sh || exit 1

# consts

KIND_IMAGE=${KIND_IMAGE:-"kindest/node:v1.25.3@sha256:f52781bc0d7a19fb6c405c2af83abfeb311f130707a0e219175677e366cc45d1"}
# K8s 1.24 or newer requires v0.10.0 as https://github.com/kubernetes-sigs/kubefed/pull/1515
KUBEFED_VER=$KUBEFEDCTL_VERSION

# main

cluster0=$PROJECT_NAME-0
cluster1=$PROJECT_NAME-1
cluster2=$PROJECT_NAME-2
cluster3=$PROJECT_NAME-3

lib::start-docker

lib::create-cluster "$cluster0" "$KIND_IMAGE"
lib::create-cluster "$cluster1" "$KIND_IMAGE"
lib::create-cluster "$cluster2" "$KIND_IMAGE"
lib::create-cluster "$cluster3" "$KIND_IMAGE"

lib::setup-kubefed "$cluster0" "$KUBEFED_VER"
lib::join-kubefed "$cluster1" "$cluster0"
lib::join-kubefed "$cluster2" "$cluster0"
lib::join-kubefed "$cluster3" "$cluster0"

"$KUBECTL" get kubefedclusters -n kube-federation-system
