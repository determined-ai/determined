#!/usr/bin/env bash

source /run/determined/task-signal-handling.sh
source /run/determined/task-logging-setup.sh

if [ -z "$DET_PYTHON_EXECUTABLE" ]; then
    export DET_PYTHON_EXECUTABLE="python3"
fi
if ! which "$DET_PYTHON_EXECUTABLE" >/dev/null 2>&1; then
    echo "error: unable to find python3 as \"$DET_PYTHON_EXECUTABLE\"" >&2
    echo "please install python3 or set the environment variable DET_PYTHON_EXECUTABLE=/path/to/python3" >&2
    exit 1
fi

set -e

"$DET_PYTHON_EXECUTABLE" -m determined.exec.prep_container --proxy

trap_and_forward_signals
if [ "$#" -eq 1 ]; then
    /bin/sh -c "$@" &
else
    "$@" &
fi
wait_and_handle_signals $!
