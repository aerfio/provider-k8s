#!/usr/bin/env bash

set -o errexit
set -o pipefail
set -o errtrace

ROOT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )/.." &> /dev/null && pwd )"
BIN_DIR="${ROOT_DIR}/bin"

BINARY_NAME="$(basename "$1")"
GOBIN=$BIN_DIR go install "${1}@${2}"
mv "$BIN_DIR/$BINARY_NAME" "$BIN_DIR/${BINARY_NAME}-${2}"
