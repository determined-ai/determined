#!/usr/bin/env bash

source /run/determined/task-setup.sh

set -e

"$DET_PYTHON_EXECUTABLE" -m determined.exec.prep_container

exec "$DET_PYTHON_EXECUTABLE" -m determined.exec.gc_checkpoints "$@"
