#!/bin/bash

source /run/determined/task-signal-handling.sh
source /run/determined/task-logging-setup.sh

set -e
set -x

STARTUP_HOOK="startup-hook.sh"
export PATH="/run/determined/pythonuserbase/bin:$PATH"
if [ -z "$DET_PYTHON_EXECUTABLE" ] ; then
    export DET_PYTHON_EXECUTABLE="python3"
fi
if ! /bin/which "$DET_PYTHON_EXECUTABLE" >/dev/null 2>&1 ; then
    echo "error: unable to find python3 as \"$DET_PYTHON_EXECUTABLE\"" >&2
    echo "please install python3 or set the environment variable DET_PYTHON_EXECUTABLE=/path/to/python3" >&2
    exit 1
fi

if [ -z "$DET_SKIP_PIP_INSTALL" ]; then
	"$DET_PYTHON_EXECUTABLE" -m pip install -q --user /opt/determined/wheels/determined*.whl

	# Install tensorboard if not already installed (for custom PyTorch images)
	"$DET_PYTHON_EXECUTABLE" -m pip install tensorboard tensorboard-plugin-profile
fi

"$DET_PYTHON_EXECUTABLE" -m determined.exec.prep_container

test -f "${STARTUP_HOOK}" && source "${STARTUP_HOOK}"

READINESS_REGEX="TensorBoard contains metrics"
TENSORBOARD_VERSION=$("$DET_PYTHON_EXECUTABLE" -c "import tensorboard; print(tensorboard.__version__)")

trap_and_forward_signals
"$DET_PYTHON_EXECUTABLE" -m determined.exec.tensorboard "$TENSORBOARD_VERSION" "$@" \
    > >(tee -p >("$DET_PYTHON_EXECUTABLE" /run/determined/check_ready_logs.py --ready-regex "$READINESS_REGEX")) &
wait_and_handle_signals $!
