#!/bin/bash

source /run/determined/task-setup.sh

set -e

"$DET_PYTHON_EXECUTABLE" -m determined.exec.prep_container --download_context_directory --resources --proxy

STARTUP_HOOK="startup-hook.sh"
set -x
test -f "${STARTUP_HOOK}" && source "${STARTUP_HOOK}"
set +x

# Do rendezvous last, to ensure all launch layers start around the same time.
"$DET_PYTHON_EXECUTABLE" -m determined.exec.prep_container --rendezvous

exec "$DET_PYTHON_EXECUTABLE" -m determined.exec.launch "$@"
