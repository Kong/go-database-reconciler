#!/bin/bash -e

function install-deepcopy-gen() {
	go install k8s.io/code-generator/cmd/deepcopy-gen
}

function run-deepcopy-gen() {
  local output_file="${1}"
  deepcopy-gen \
    --output-file zz_generated.deepcopy.go \
    --go-header-file scripts/header-template.go.tmpl \
    "${output_file}"
}
