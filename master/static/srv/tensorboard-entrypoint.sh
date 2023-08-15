#!/bin/bash

source /run/determined/task-signal-handling.sh
source /run/determined/task-logging-setup.sh

set -e

STARTUP_HOOK="startup-hook.sh"
export PATH="/run/determined/pythonuserbase/bin:$PATH"
if [ -z "$DET_PYTHON_EXECUTABLE" ]; then
    export DET_PYTHON_EXECUTABLE="python3"
fi

if [ -z "$DET_SKIP_PIP_INSTALL" ]; then
    # Install tensorboard if not already installed (for custom PyTorch images)
    "$DET_PYTHON_EXECUTABLE" -m pip install tensorboard tensorboard-plugin-profile
fi

"$DET_PYTHON_EXECUTABLE" -m determined.exec.prep_container --proxy

set -x
test -f "${STARTUP_HOOK}" && source "${STARTUP_HOOK}"
set +x

READINESS_REGEX="TensorBoard contains metrics"
WAITING_REGEX="TensorBoard waits on metrics"
TENSORBOARD_VERSION=$("$DET_PYTHON_EXECUTABLE" -c "import tensorboard; print(tensorboard.__version__)")

trap_and_forward_signals
"$DET_PYTHON_EXECUTABLE" -m determined.exec.tensorboard "$TENSORBOARD_VERSION" "$@" \
    > >(tee -p >("$DET_PYTHON_EXECUTABLE" /run/determined/check_ready_logs.py --ready-regex "$READINESS_REGEX" --waiting-regex "$WAITING_REGEX")) &
wait_and_handle_signals $!
