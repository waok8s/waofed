#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

SCRIPT=$(realpath "$0")
PROJECT_ROOT=$(dirname "$(dirname "$SCRIPT")")
PROJECT_NAME=$(basename "$PROJECT_ROOT")

KIND_IMAGE=${KIND_IMAGE:-"kindest/node:v1.25.3@sha256:f52781bc0d7a19fb6c405c2af83abfeb311f130707a0e219175677e366cc45d1"}
CERT_MANAGER_YAML=${CERT_MANAGER_YAML:-"https://github.com/cert-manager/cert-manager/releases/download/v1.10.0/cert-manager.yaml"}
METRICS_SERVER_YAML=${METRICS_SERVER_YAML:-"https://github.com/kubernetes-sigs/metrics-server/releases/download/v0.6.1/components.yaml"}
METRICS_SERVER_PATCH=${METRICS_SERVER_PATCH:-'''[{"op":"add","path":"/spec/template/spec/containers/0/args/-","value":"--kubelet-insecure-tls"}]'''}
# K8s 1.24 or newer requires v0.10.0 as https://github.com/kubernetes-sigs/kubefed/pull/1515
KUBEFED_VER=${KUBEFED_VER:-"0.10.0"}

# Usage: create-cluster <name> <kind_image>
function create-cluster {
    local kind_name=$1
    local name=kind-$1
    local kind_image=$2

    kind delete cluster --name "$kind_name"

    kind create cluster --name "$kind_name" --image="$kind_image" --config=- <<EOF
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
- role: worker
EOF

    local docker_ip
    docker_ip=$(docker inspect --format='{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' "${kind_name}-control-plane")
    kubectl config set-cluster "$name" --server="https://${docker_ip}:6443"

    kubectl apply -f "$CERT_MANAGER_YAML"
    kubectl apply -f "$METRICS_SERVER_YAML"
    kubectl patch -n kube-system deployment metrics-server --type=json -p "$METRICS_SERVER_PATCH"
}

# Usage: setup-kubefed <hq_name> <ver>
function setup-kubefed {
    local hq_name=kind-$1
    local ver=$2

    which helm

    kubectl config use-context "$hq_name"
    helm repo add kubefed-charts https://raw.githubusercontent.com/kubernetes-sigs/kubefed/master/charts
    helm --namespace kube-federation-system upgrade -i kubefed kubefed-charts/kubefed --version=v"$ver" --create-namespace
}

# Usage: join-kubefed <member_name> <hq_name>
function join-kubefed {
    local member_name=kind-$1
    local hq_name=kind-$2

    which kubefedctl

    kubefedctl join "$member_name" --cluster-context="$member_name" --host-cluster-context="$hq_name" --v=5

    kubectl config use-context "$hq_name"
}

# main

cluster0=$PROJECT_NAME-test-0

cd "$PROJECT_ROOT"

sudo systemctl start docker || sudo service docker start || true
sleep 1

# https://kind.sigs.k8s.io/docs/user/known-issues/#pod-errors-due-to-too-many-open-files
sudo sysctl fs.inotify.max_user_watches=524288
sudo sysctl fs.inotify.max_user_instances=512

create-cluster "$cluster0" "$KIND_IMAGE"

setup-kubefed "$cluster0" "$KUBEFED_VER"
join-kubefed "$cluster0" "$cluster0"

kubectl get kubefedclusters -n kube-federation-system
