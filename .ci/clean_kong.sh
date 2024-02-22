#!/bin/bash

set -e

NETWORK_NAME=deck-test

PG_CONTAINER_NAME=pg
GATEWAY_CONTAINER_NAME=kong

if [[ $(docker ps -a | grep ${GATEWAY_CONTAINER_NAME}) != "" ]]; then
    echo "remove container ${GATEWAY_CONTAINER_NAME}"
    docker rm -f  ${GATEWAY_CONTAINER_NAME}
else
    echo "container ${GATEWAY_CONTAINER_NAME} not found, skip removing"
fi

if [[ $(docker ps -a | grep ${PG_CONTAINER_NAME}) != "" ]]; then
    echo "remove container ${PG_CONTAINER_NAME}"
    docker rm -f  ${PG_CONTAINER_NAME}
else
    echo "container ${PG_CONTAINER_NAME} not found, skip removing"
fi

if [[ $(docker network ls | grep $NETWORK_NAME) != "" ]]; then
    echo "remove docker network ${NETWORK_NAME}"
    docker network rm ${NETWORK_NAME}
else
    echo "docker network ${NETWORK_NAME} does not exist, skip removing"
fi
