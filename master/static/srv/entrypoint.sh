#!/usr/bin/env bash

set -e
set -x

WORKING_DIR="/run/determined/workdir"
STARTUP_HOOK="./startup-hook.sh"
export PATH="/run/determined/pythonuserbase/bin:$PATH"

python3.6 -m pip install --user /opt/determined/wheels/determined*.whl

cd ${WORKING_DIR} && test -f "${STARTUP_HOOK}" && source "${STARTUP_HOOK}"
exec python3.6 -m determined.exec.harness "$@"
