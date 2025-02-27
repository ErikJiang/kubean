#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

API_REPO_ROOT=$(pwd)

bash "$API_REPO_ROOT/hack/update-codegen.sh"
bash "$API_REPO_ROOT/hack/update-crdgen.sh"

go mod tidy
