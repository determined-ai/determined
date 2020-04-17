#!/bin/bash

set -e

export PATH="/run/determined/pythonuserbase/bin:$PATH"

STARTUP_HOOK=${STARTUP_HOOK_SCRIPT:-startup-hook.sh}

# Source `startup-hook.sh` before running tensorboard.
source <(python3.6 -m determined.exec.prepare_env ${STARTUP_HOOK})

python3.6 -m pip install --user /opt/determined/wheels/determined*.whl

# S3 backed experiments must have AWS_REGION set in the environment.
eval "$(python3.6 -m determined.exec.tensorboard s3)"

python3.6 -m determined.exec.tensorboard service_ready &

tensorboard --port=${TENSORBOARD_PORT} --path_prefix="/proxy/${DET_TASK_ID}" $@
