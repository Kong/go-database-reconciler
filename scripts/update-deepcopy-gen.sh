#!/bin/bash -e

set -x

SCRIPT_DIR="$(dirname $0)"
source ${SCRIPT_DIR}/_lib.sh

install-deepcopy-gen

run-deepcopy-gen ./pkg/konnect
run-deepcopy-gen ./pkg/file
