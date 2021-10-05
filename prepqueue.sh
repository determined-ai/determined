#!/bin/bash

det e create $PROJECT_ROOT/e2e_tests/tests/fixtures/no_op/adaptive.yaml $PROJECT_ROOT/e2e_tests/tests/fixtures/no_op > /dev/null &

det shell start > /dev/null &
sleep 1

det tensorboard start 1 -t 1 > /dev/null &
sleep 1

det notebook start > /dev/null &
sleep 1

curl-da.sh "${DET_MASTER}/api/v1/resource-pools/queues?resource_pool=default" -s | jq
