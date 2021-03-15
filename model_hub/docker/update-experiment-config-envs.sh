#!/bin/bash
set -e

DOCKER_IMAGE=$1
echo $DOCKER_IMAGE

configs=( $(find ./examples -type f -name "*.yaml") )
for c in ${configs[@]};
do
    sed -i "s/model-hub-transformers:.*/${DOCKER_IMAGE}/" $c
done
