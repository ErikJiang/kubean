#!/usr/bin/env bash

# Copyright 2025 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -o errexit
set -o nounset
set -o pipefail

SCRIPT_ROOT=$(dirname "${BASH_SOURCE[0]}")/..
# CODEGEN_PKG=${CODEGEN_PKG:-$(cd "${SCRIPT_ROOT}"; ls -d -1 ./vendor/k8s.io/code-generator 2>/dev/null || echo ../code-generator)}
MODULE_NAME=github.com/kubean-io/kubean-api
BOILERPLATE=${SCRIPT_ROOT}/hack/boilerplate.go.txt

CODEGEN_VERSION="v0.33.0-alpha.3"
# go get k8s.io/code-generator@${CODEGEN_VERSION}
go mod download k8s.io/code-generator@${CODEGEN_VERSION}
CODEGEN_PKG="$(echo `go env GOPATH`/pkg/mod/k8s.io/code-generator@${CODEGEN_VERSION})"

echo ">>> using codegen: ${CODEGEN_PKG}"

# Source the kube_codegen.sh script
source "${CODEGEN_PKG}/kube_codegen.sh"

echo "Generating code for ${MODULE_NAME}..."

# Create boilerplate file if it doesn't exist
touch "${BOILERPLATE}"

# 清理现有的生成文件
rm -f ./apis/cluster/v1alpha1/zz_generated*
rm -f ./apis/clusteroperation/v1alpha1/zz_generated*
rm -f ./apis/localartifactset/v1alpha1/zz_generated*
rm -f ./apis/manifest/v1alpha1/zz_generated*

# Generate helpers (deepcopy, defaulter, etc.)
GOFLAGS=-mod=mod kube::codegen::gen_helpers \
  --boilerplate "${BOILERPLATE}" \
  ./apis

# export KUBE_VERBOSE=9
# Generate registers
GOFLAGS=-mod=mod kube::codegen::gen_register \
  --boilerplate "${BOILERPLATE}" \
  ./apis

# Generate client code
GOFLAGS=-mod=mod kube::codegen::gen_client \
  --with-watch \
  --with-applyconfig \
  --output-pkg "${MODULE_NAME}/client" \
  --output-dir ./client \
  --boilerplate "${BOILERPLATE}" \
  ./apis

# Clean up
rm -rf "${BOILERPLATE}"

echo "Code generation complete."
