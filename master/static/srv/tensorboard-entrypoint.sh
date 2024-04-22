#!/bin/bash

source /run/determined/task-setup.sh

set -e

if [ -z "$DET_SKIP_PIP_INSTALL" ]; then
    # Install tensorboard if not already installed (for custom PyTorch images)
    "$DET_PYTHON_EXECUTABLE" -m pip install tensorboard tensorboard-plugin-profile
fi

"$DET_PYTHON_EXECUTABLE" -m determined.exec.prep_container --proxy --download_context_directory

STARTUP_HOOK="startup-hook.sh"
set -x
test -f "${TCD_STARTUP_HOOK}" && source "${TCD_STARTUP_HOOK}"
test -f "${STARTUP_HOOK}" && source "${STARTUP_HOOK}"
set +x

READINESS_REGEX="TensorBoard contains metrics"
WAITING_REGEX="TensorBoard waits on metrics"
TENSORBOARD_VERSION=$("$DET_PYTHON_EXECUTABLE" -c "import tensorboard; print(tensorboard.__version__)")

"$DET_PYTHON_EXECUTABLE" -m determined.exec.tensorboard "$TENSORBOARD_VERSION" "$@" \
    > >(tee -p >("$DET_PYTHON_EXECUTABLE" /run/determined/check_ready_logs.py --ready-regex "$READINESS_REGEX" --waiting-regex "$WAITING_REGEX"))
