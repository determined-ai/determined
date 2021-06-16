#!/bin/bash

set -e
set -x

WORKING_DIR="/run/determined/workdir"
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

TENSORBOARD_VERSION=$(pip show tensorboard | grep Version | sed "s/[^:]*: *//")
TENSORBOARD_VERSION_MAJOR=$( echo "$TENSORBOARD_VERSION" | sed " s/[.].*//")

"$DET_PYTHON_EXECUTABLE" -m pip install -q --user /opt/determined/wheels/determined*.whl

cd ${WORKING_DIR} && test -f "${STARTUP_HOOK}" && source "${STARTUP_HOOK}"

if [ "$TENSORBOARD_VERSION_MAJOR" == 2 ]; then
  "$DET_PYTHON_EXECUTABLE" -m pip install tensorboard-plugin-profile
  exec "$DET_PYTHON_EXECUTABLE" -m determined.exec.tensorboard --bind_all --logdir_spec "$@"
else
  exec "$DET_PYTHON_EXECUTABLE" -m determined.exec.tensorboard --logdir "$@"
fi
