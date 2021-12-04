#!/bin/bash

det e create $PROJECT_ROOT/examples/tutorials/mnist_pytorch/const.yaml $PROJECT_ROOT/examples/tutorials/mnist_pytorch > /dev/null &

det shell start > /dev/null &
sleep 0.5

det tensorboard start 1 -t 1 > /dev/null &
sleep 0.5

det notebook start > /dev/null &
sleep 0.5

curl-da.sh "${DET_MASTER}/api/v1/resource-pools/queues" -s | jq
