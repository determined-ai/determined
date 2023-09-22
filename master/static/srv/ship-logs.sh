#!/usr/bin/env bash

# Ideally, we'd like to make ship_logs.py the entrypoint in task containers, so
# it could capture all logs from any process in the process tree.
#
# But we can't actually set it as the entrypoint because we don't know how to
# call python in the container until we're inside the container.
#
# So ship-logs.sh runs inside the container, figures out how to call python, and
# then calls ship_logs.py.

set -e

if [ -z "$DET_PYTHON_EXECUTABLE" ]; then
    export DET_PYTHON_EXECUTABLE="python3"
fi

ship_logs="$1"
shift

exec "$DET_PYTHON_EXECUTABLE" "$ship_logs" "$@"
