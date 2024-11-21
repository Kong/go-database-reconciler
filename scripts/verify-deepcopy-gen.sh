#!/bin/bash -e

set -x

SCRIPT_DIR="$(dirname $0)"
source ${SCRIPT_DIR}/_lib.sh

TMP_DIR=$(mktemp -d)
trap "rm -rf $TMP_DIR" EXIT

function run-deepcopy-and-diff() {
  local pkg="${1}"

  cp ./pkg/${pkg}/zz_generated.deepcopy.go ${TMP_DIR}/${pkg}.zz_generated.deepcopy.go
  run-deepcopy-gen ./pkg/${pkg}

  diff -Naur ./pkg/${pkg}/zz_generated.deepcopy.go \
    ${TMP_DIR}/${pkg}.zz_generated.deepcopy.go
}

install-deepcopy-gen

run-deepcopy-and-diff konnect
run-deepcopy-and-diff file
