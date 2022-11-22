#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

SCRIPT=$(realpath "$0")
PROJECT_ROOT=$(dirname "$(dirname "$SCRIPT")")
PROJECT_NAME=$(basename "$PROJECT_ROOT")

cluster0=$PROJECT_NAME-test-0

cd "$PROJECT_ROOT"

kubectl config use-context kind-"$cluster0"

make test-on-existing-cluster
