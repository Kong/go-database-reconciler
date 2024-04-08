#!/bin/bash

set -euo pipefail

source ./.ci/lib.sh

KONG_IMAGE=${KONG_IMAGE?KONG_IMAGE is required to be set}

initNetwork
initDb
initMigrations "$KONG_IMAGE"

GATEWAY_CONTAINER_NAME=kong

# Start Kong Gateway
docker run \
    --detach \
    --name $GATEWAY_CONTAINER_NAME \
    "${DOCKER_ARGS[@]}" \
    -e "KONG_ADMIN_LISTEN=0.0.0.0:8001, 0.0.0.0:8444 ssl" \
    -p 8000:8000 \
    -p 8443:8443 \
    -p 127.0.0.1:8001:8001 \
    -p 127.0.0.1:8444:8444 \
    "$KONG_IMAGE"

waitContainer "$GATEWAY_CONTAINER_NAME" kong health
