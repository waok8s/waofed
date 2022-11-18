#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

SCRIPT=$(realpath "$0")
PROJECT_ROOT=$(dirname "$(dirname "$SCRIPT")")
PROJECT_NAME=$(basename "$PROJECT_ROOT")

VERSION=$(git describe --tags --match "v*")
IMG=$PROJECT_NAME-controller:$VERSION

KIND_CLUSTER_NAME=$PROJECT_NAME-0
CLUSTER_NAME=kind-$KIND_CLUSTER_NAME

cd "$PROJECT_ROOT"
kubectl config use-context "$CLUSTER_NAME"

make
make docker-build IMG="$IMG"
kind load docker-image "$IMG" -n "$KIND_CLUSTER_NAME"

make undeploy || true
make deploy IMG="$IMG"
