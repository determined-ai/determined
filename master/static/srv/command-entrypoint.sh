#!/usr/bin/env bash

source /run/determined/task-signal-handling.sh
source /run/determined/task-logging-setup.sh

set -e

if [ -z "$DET_PYTHON_EXECUTABLE" ]; then
    export DET_PYTHON_EXECUTABLE="python3"
fi

# In order to be able to use a proxy when running a command, Python must be
# available in the container, and the "determined*.whl" must be installed,
# which contains the "determined/exec/prep_container.py" script that's needed
# to register the proxy with the Determined master.
"$DET_PYTHON_EXECUTABLE" -m determined.exec.prep_container --proxy --download_context_directory

trap_and_forward_signals
if [ "$#" -eq 1 ]; then
    /bin/sh -c "$@" &
else
    "$@" &
fi
wait_and_handle_signals $!
