#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "${SCRIPT_DIR}")"
BIN_DIR="${PROJECT_ROOT}/hack/bin"

CONTROLLER_GEN_PKG=sigs.k8s.io/controller-tools/cmd/controller-gen
CONTROLLER_GEN_VER=v0.17.2

mkdir -p "${BIN_DIR}"
GOBIN="${BIN_DIR}" go install "${CONTROLLER_GEN_PKG}"@"${CONTROLLER_GEN_VER}"

# Unify the crds used by helm chart and the installation scripts
"${BIN_DIR}/controller-gen" crd paths=./apis/... output:crd:dir=./charts/crds

for f in ./charts/crds/* ; do
  ## f: "./charts/crds/kubean.io_clusteroperations.yaml"
  sed '/^[[:blank:]]*$/d' "$f" > "$f.tmp" ## remove blank
  mv "$f.tmp"  "$f"
done

cp -r charts/crds ../charts/kubean

rm -rf "${BIN_DIR}"

echo "CRD generation complete."