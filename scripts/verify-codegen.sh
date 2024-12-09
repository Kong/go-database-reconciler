#!/bin/bash -e

set -x

SCHEMA_FILE_NAME="kong_json_schema.json"
SOURCE_FILE_PATH="pkg/file/${SCHEMA_FILE_NAME}"
TMP_SCHEMA_FILE_PATH="/tmp/${SCHEMA_FILE_NAME}"

cp "${SOURCE_FILE_PATH}" "${TMP_SCHEMA_FILE_PATH}"
go generate ./...

diff -u "${TMP_SCHEMA_FILE_PATH}" "${SOURCE_FILE_PATH}"
