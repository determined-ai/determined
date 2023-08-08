#!/usr/bin/env bash

source /run/determined/task-signal-handling.sh
source /run/determined/task-logging-setup.sh

set -e

if [ -z "$DET_PYTHON_EXECUTABLE" ]; then
    export DET_PYTHON_EXECUTABLE="python3"
fi

# Check if Python is available in the container.
if which "$DET_PYTHON_EXECUTABLE" >/dev/null 2>&1; then
    # In order to be able to use a proxy when running a command, Python must be
    # available in the container, and the "determined*.whl" must be installed,
    # which contains the "determined/exec/prep_container.py" script that's needed
    # to register the proxy with the Determined master.
    if [ -z "$DET_SKIP_PIP_INSTALL" ]; then
        "$DET_PYTHON_EXECUTABLE" -m pip install -q --user /opt/determined/wheels/determined*.whl
    fi

    "$DET_PYTHON_EXECUTABLE" -m determined.exec.prep_container --proxy
else
    # Not all commands will require that Python be installed in the container.
    # Some of the e2e tests use an image that does not contain Python, so we
    # don't want to fail the test in those cases.  Therefore, issue a warning,
    # but do not exit the script.
    echo "warn: unable to find python3 as \"$DET_PYTHON_EXECUTABLE\"" >&2
    echo "please install python3 or set the environment variable DET_PYTHON_EXECUTABLE=/path/to/python3" >&2
fi

trap_and_forward_signals
if [ "$#" -eq 1 ]; then
    /bin/sh -c "$@" &
else
    "$@" &
fi
wait_and_handle_signals $!
