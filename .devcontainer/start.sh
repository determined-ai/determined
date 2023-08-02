#!/usr/bin/env bash
set -xeuo pipefail

STAMP=${HOME}/.determined-setup.stamp
if [ ! -f ${STAMP} ]; then
    .devcontainer/setup.sh
    touch ${STAMP}
fi

LOG=${HOME}/determined-server.log
touch ${LOG}
nohup .devcontainer/server.sh >>${LOG} &
tail -f ${LOG}
