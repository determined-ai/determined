#!/usr/bin/env bash

set -e

export PATH="/run/determined/pythonuserbase/bin:$PATH"
if [ -z "$DET_PYTHON_EXECUTABLE" ] ; then
    export DET_PYTHON_EXECUTABLE="python3"
fi
if ! /bin/which "$DET_PYTHON_EXECUTABLE" >/dev/null 2>&1 ; then
    echo "error: unable to find python3 as \"$DET_PYTHON_EXECUTABLE\"" >&2
    echo "please install python3 or set the environment variable DET_PYTHON_EXECUTABLE=/path/to/python3" >&2
    exit 1
fi


"$DET_PYTHON_EXECUTABLE" -m pip install -q --user /opt/determined/wheels/determined*.whl

exec "$DET_PYTHON_EXECUTABLE" -m determined.exec.gc_checkpoints "$@"
