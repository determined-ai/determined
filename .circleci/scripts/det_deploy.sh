#!/usr/bin/env bash

det deploy $@
res=$?
if [ $res -ne 0 ] && [ "${CLUSTER_ID}" != "" ]; then
    det deploy $1 down --cluster-id ${CLUSTER_ID}
fi
exit $res
