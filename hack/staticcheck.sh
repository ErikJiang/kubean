#!/usr/bin/env bash

# Copyright 2023 Authors of kubean-io
# SPDX-License-Identifier: Apache-2.0

set -o errexit
set -o nounset
set -o pipefail

REPO_ROOT=$(dirname "${BASH_SOURCE[0]}")/..
GOLANGCI_LINT_VER="v2.11.4"

cd "${REPO_ROOT}"
source "hack/util.sh"

# Build golangci-lint with the active Go toolchain so the binary stays compatible
# with the Go version declared by this repository.
GOLANGCI_LINT_BIN="$(go env GOPATH)/bin/golangci-lint"
GOBIN="$(go env GOPATH)/bin" go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@${GOLANGCI_LINT_VER}

if "${GOLANGCI_LINT_BIN}" run --fix --verbose; then
  echo 'Congratulations!  All Go source files have passed staticcheck.'
else
  echo # print one empty line, separate from warning messages.
  echo 'Please review the above warnings.'
  echo 'If the above warnings do not make sense, feel free to file an issue.'
  exit 1
fi
