#!/usr/bin/env bash

# scripts must be run from project root
. hack/1-bin.sh || exit 1

# consts

CERT_MANAGER_YAML=${CERT_MANAGER_YAML:-"https://github.com/cert-manager/cert-manager/releases/download/v1.10.0/cert-manager.yaml"}
METRICS_SERVER_YAML=${METRICS_SERVER_YAML:-"https://github.com/kubernetes-sigs/metrics-server/releases/download/v0.6.1/components.yaml"}
METRICS_SERVER_PATCH=${METRICS_SERVER_PATCH:-'''[{"op":"add","path":"/spec/template/spec/containers/0/args/-","value":"--kubelet-insecure-tls"}]'''}

WAO_ESTIMATOR_YAML=${WAO_ESTIMATOR_YAML:-"https://github.com/Nedopro2022/wao-estimator/releases/download/v0.1.0/wao-estimator.yaml"}

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

    "$KUBECTL" apply -f "$METRICS_SERVER_YAML"
    "$KUBECTL" patch -n kube-system deployment metrics-server --type=json -p "$METRICS_SERVER_PATCH"

    "$KUBECTL" apply -f "$CERT_MANAGER_YAML"
    "$KUBECTL" wait deploy -ncert-manager cert-manager --for=condition=Available=True --timeout=60s
    "$KUBECTL" wait deploy -ncert-manager cert-manager-cainjector --for=condition=Available=True --timeout=60s
    "$KUBECTL" wait deploy -ncert-manager cert-manager-webhook --for=condition=Available=True --timeout=60s
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
    "$KUBECTL" wait kfc -nkube-federation-system "$member_name" --for=condition=ready --timeout=60s
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

# Usage: lib::install-wao-estimator <cluster_name>
function lib::install-wao-estimator {
    local cluster_name=kind-$1
    test -s "$KUBECTL"
    "$KUBECTL" apply --context "$cluster_name" -f "$WAO_ESTIMATOR_YAML"
}

# Usage: lib::start-wao-estimator <cluster_name> <estimator_yaml> <port>
function lib::start-wao-estimator {
    local cluster_name=kind-$1
    local estimator_yaml=$2
    local port=$3

    test -s "$KUBECTL"
    test -s "$estimator_yaml"
    test -s "$ESTIMATOR_CLI"

    "$KUBECTL" --context "$cluster_name" apply -f "$estimator_yaml"
    "$KUBECTL" --context "$cluster_name" port-forward -n wao-estimator-system \
        svc/wao-estimator-controller-manager-estimator-service "$port":5656 &
    
    sleep 1

    "$ESTIMATOR_CLI" -v -a "http://localhost:${port}" pc
}
