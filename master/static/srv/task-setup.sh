#!/usr/bin/env bash

STDOUT_FILE=/run/determined/train/logs/stdout.log
STDERR_FILE=/run/determined/train/logs/stderr.log

mkdir -p "$(dirname "$STDOUT_FILE")" "$(dirname "$STDERR_FILE")"

# Create symbolic links from well-known files to this process's STDOUT and
# STDERR. Anything written to those files will be inserted into the output
# streams of this process, allowing distributed training logs to route through
# individual containers rather than all going through SSH back to agent 0.
ln -sf /proc/$$/fd/1 "$STDOUT_FILE"
ln -sf /proc/$$/fd/2 "$STDERR_FILE"

export PATH="/run/determined/pythonuserbase/bin:$PATH"
if [ -z "$DET_PYTHON_EXECUTABLE" ]; then
    export DET_PYTHON_EXECUTABLE="python3"
fi

if ! "$DET_PYTHON_EXECUTABLE" --version >/dev/null 2>&1; then
    echo "{\"log\": \"error: unable to find python3 as '$DET_PYTHON_EXECUTABLE'\n\", \"timestamp\": \"$(date --rfc-3339=seconds)\"}" >&2
    echo "{\"log\": \"please install python3 or set the environment variable DET_PYTHON_EXECUTABLE=/path/to/python3\n\", \"timestamp\": "$(date --rfc-3339=seconds)"}" >&2
    exit 1
fi

if [ -z "$DET_SKIP_PIP_INSTALL" ]; then
    "$DET_PYTHON_EXECUTABLE" -m pip install -q --user /opt/determined/wheels/determined*.whl
else
    if ! "$DET_PYTHON_EXECUTABLE" -c "import determined" >/dev/null 2>&1; then
        echo "{\"log\": \"error: unable run without determined package\n\", \"timestamp\": \"$(date --rfc-3339=seconds)\"}" >&2
        exit 1
    fi
fi

if [ "$DET_RESOURCES_TYPE" == "slurm-job" ]; then
    # Each container sends the Determined Master a notification that it's
    # running, so that the Determined Master knows whether to set the state
    # of the experiment to "Pulling", meaning some nodes are pulling down
    # the image, or "Running", meaning that all containers are running.
    "$DET_PYTHON_EXECUTABLE" -m determined.exec.prep_container --notify_container_running
fi
