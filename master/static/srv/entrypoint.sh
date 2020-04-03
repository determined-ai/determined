#!/usr/bin/env bash

set -e
set -x

WORKING_DIR="/run/determined/workdir"
STARTUP_HOOK="./startup-hook.sh"

python3.6 -m pip install --user --upgrade --find-links /opt/determined/wheels determined
cd ${WORKING_DIR} && test -f "${STARTUP_HOOK}" && source "${STARTUP_HOOK}"
exec python3.6 -m determined.exec.harness "$@"
