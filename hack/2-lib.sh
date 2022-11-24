#!/usr/bin/env bash

# scripts must be run from project root
. hack/1-bin.sh || exit 1

# consts

CERT_MANAGER_YAML=${CERT_MANAGER_YAML:-"https://github.com/cert-manager/cert-manager/releases/download/v1.10.0/cert-manager.yaml"}
METRICS_SERVER_YAML=${METRICS_SERVER_YAML:-"https://github.com/kubernetes-sigs/metrics-server/releases/download/v0.6.1/components.yaml"}
METRICS_SERVER_PATCH=${METRICS_SERVER_PATCH:-'''[{"op":"add","path":"/spec/template/spec/containers/0/args/-","value":"--kubelet-insecure-tls"}]'''}

# libs

# Usage: lib::create-cluster <name> <kind_image>
function lib::create-cluster {
    local kind_name=$1
    local name=kind-$1
    local kind_image=$2

    test -s "$KIND"
    test -s "$KUBECTL"

    "$KIND" delete cluster --name "$kind_name"

    "$KIND" create cluster --name "$kind_name" --image="$kind_image" --config=- <<EOF
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
- role: worker
EOF

    local docker_ip
    docker_ip=$(docker inspect --format='{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' "${kind_name}-control-plane")
    "$KUBECTL" config set-cluster "$name" --server="https://${docker_ip}:6443"

    "$KUBECTL" apply -f "$CERT_MANAGER_YAML"
    "$KUBECTL" apply -f "$METRICS_SERVER_YAML"
    "$KUBECTL" patch -n kube-system deployment metrics-server --type=json -p "$METRICS_SERVER_PATCH"
}

# Usage: lib::setup-kubefed <hq_name> <ver>
function lib::setup-kubefed {
    local hq_name=kind-$1
    local ver=$2

    test -s "$KUBECTL"
    test -s "$HELM"
    
    "$KUBECTL" config use-context "$hq_name"
    "$HELM" repo add kubefed-charts https://raw.githubusercontent.com/kubernetes-sigs/kubefed/master/charts
    "$HELM" --namespace kube-federation-system upgrade -i kubefed kubefed-charts/kubefed --version=v"$ver" --create-namespace
}

# Usage: lib::join-kubefed <member_name> <hq_name>
function lib::join-kubefed {
    local member_name=kind-$1
    local hq_name=kind-$2

    test -s "$KUBEFEDCTL"

    "$KUBEFEDCTL" join "$member_name" --cluster-context="$member_name" --host-cluster-context="$hq_name" --v=5

    "$KUBECTL" config use-context "$hq_name"
}

# Usage: lib::start-docker
function lib::start-docker {

    set +o nounset
    if [ "$CI" == "true" ]; then return 0; fi
    set -o nounset

    sudo systemctl start docker || sudo service docker start || true
    sleep 1

    # https://kind.sigs.k8s.io/docs/user/known-issues/#pod-errors-due-to-too-many-open-files
    sudo sysctl fs.inotify.max_user_watches=524288 || true
    sudo sysctl fs.inotify.max_user_instances=512 || true
}
