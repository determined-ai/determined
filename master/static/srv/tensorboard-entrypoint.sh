#!/bin/bash

set -e
set -x

WORKING_DIR="/run/determined/workdir"
STARTUP_HOOK="startup-hook.sh"
export PATH="/run/determined/pythonuserbase/bin:$PATH"

python3.6 -m pip install --user /opt/determined/wheels/determined*.whl

cd ${WORKING_DIR} && test -f "${STARTUP_HOOK}" && source "${STARTUP_HOOK}"

# S3 backed experiments must have AWS_REGION set in the environment.
eval "$(python3.6 -m determined.exec.tensorboard s3)"

python3.6 -m determined.exec.tensorboard service_ready &

tensorboard --port=${TENSORBOARD_PORT} --path_prefix="/proxy/${DET_TASK_ID}" $@
