#!/usr/bin/env bash

source /run/determined/task-setup.sh

set -e

export PATH="/run/determined/pythonuserbase/bin:$PATH"
if [ -z "$DET_PYTHON_EXECUTABLE" ]; then
    export DET_PYTHON_EXECUTABLE="python3"
fi

"$DET_PYTHON_EXECUTABLE" -m determined.exec.prep_container

exec "$DET_PYTHON_EXECUTABLE" -m determined.exec.gc_checkpoints "$@"
