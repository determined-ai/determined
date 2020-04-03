#!/bin/bash

set -e

STARTUP_HOOK=${STARTUP_HOOK_SCRIPT:-startup-hook.sh}

# Source `startup-hook.sh` before running tensorboard.
source <(python3.6 -m determined.exec.prepare_env ${STARTUP_HOOK})

python3.6 -m pip install --upgrade --find-links /opt/determined/wheels determined

# S3 backed experiments must have AWS_REGION set in the environment.
eval "$(python3.6 -m determined.exec.tensorboard s3)"

python3.6 -m determined.exec.tensorboard service_ready &

tensorboard --port=${TENSORBOARD_PORT} --path_prefix="/proxy/${DET_TASK_ID}" $@
