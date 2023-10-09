#!/usr/bin/env bash

det deploy "$@"
status=$?
# if we tried to bring up a cluster and it failed, recover by trying to bring the cluster down
if [ $status -ne 0 ] && [ "$2" == "up" ] && [ "${CLUSTER_ID}" != "" ]; then
    echo "CI error: det deploy $1 up failed, cleaning up cluster ${CLUSTER_ID}" >&2
    det deploy $1 down --cluster-id ${CLUSTER_ID} --yes
fi
exit $status
