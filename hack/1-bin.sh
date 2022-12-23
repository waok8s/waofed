#!/usr/bin/env bash

# scripts must be run from project root
. hack/0-env.sh || exit 1

### main ###

test -s "$KIND" || GOBIN="$LOCALBIN" go install sigs.k8s.io/kind@"$KIND_VERSION"
test -s "$KUBECTL" || (mkdir -p "$KUBECTL_DIR" ; curl -L https://dl.k8s.io/release/"$KUBECTL_VERSION"/bin/linux/amd64/kubectl > "$KUBECTL" ; chmod +x "$KUBECTL")
test -s "$KUBEFEDCTL" || (curl -L https://github.com/kubernetes-sigs/kubefed/releases/download/v"$KUBEFEDCTL_VERSION"/kubefedctl-"$KUBEFEDCTL_VERSION"-linux-amd64.tgz | tar -zxvf  - -C "$LOCALBIN" ; chmod +x "$KUBEFEDCTL")
test -s "$HELM" || (mkdir -p "$HELM_DIR" ; curl -L https://get.helm.sh/helm-"$HELM_VERSION"-linux-amd64.tar.gz | tar -zxvf - -C "$HELM_DIR" ; chmod +x "$HELM")

test -s "$ESTIMATOR_CLI" || (curl -L https://github.com/Nedopro2022/wao-estimator/releases/download/"$WAO_ESTIMATOR_VERSION"/estimator-cli > "$ESTIMATOR_CLI" ; chmod +x "$ESTIMATOR_CLI")

echo -e "= version info ="

"$KIND" version
"$KUBECTL" version --client
"$KUBEFEDCTL" version
"$HELM" version

echo -e "================\n"
