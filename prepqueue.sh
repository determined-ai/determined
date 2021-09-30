#!/bin/bash

det e create $PROJECT_ROOT/e2e_tests/tests/fixtures/no_op/adaptive.yaml $PROJECT_ROOT/e2e_tests/tests/fixtures/no_op > /dev/null &

det shell start > /dev/null &

det tensorboard start 1 -t 1 > /dev/null &

det notebook start > /dev/null &

sleep 0.5
curl-da.sh "${DET_MASTER}/api/v1/resource-pools/queues?resource_pools=default" -s | jq
