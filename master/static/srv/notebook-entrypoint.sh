#!/usr/bin/env bash

STDOUT_FILE=/run/determined/train/logs/stdout.log
STDERR_FILE=/run/determined/train/logs/stderr.log

mkdir -p "$(dirname "$STDOUT_FILE")" "$(dirname "$STDERR_FILE")"

# Explaination for this is found in ./entrypoint.sh.
ln -sf /proc/$$/fd/1 "$STDOUT_FILE"
ln -sf /proc/$$/fd/2 "$STDERR_FILE"
if [ -n "$DET_K8S_LOG_TO_FILE" ]; then
    STDOUT_ROTATE_DIR="$STDOUT_FILE-rotate"
    STDERR_ROTATE_DIR="$STDERR_FILE-rotate"
    mkdir -p -m 755 $STDOUT_ROTATE_DIR
    mkdir -p -m 755 $STDERR_ROTATE_DIR
    exec > >(multilog n2 "$STDOUT_ROTATE_DIR")  2> >(multilog n2 "$STDERR_ROTATE_DIR")
fi

set -e

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

# If HOME is not explicitly set for a container, libcontainer (Docker) will
# try to guess it by reading /etc/password directly, which will not work with
# our libnss_determined plugin (or any user-defined NSS plugin in a container).
# The default is "/", but HOME must be a writable location for distributed
# training, so we try to query the user system for a valid HOME, or default to
# the working directory otherwise.
if [ "$HOME" = "/" ] ; then
    HOME="$(set -o pipefail; getent passwd "$(whoami)" | cut -d: -f6)" || HOME="$PWD"
    export HOME
fi

# Use user's preferred SHELL in JupyterLab terminals.
SHELL="$(set -o pipefail; getent passwd "$(whoami)" | cut -d: -f7)" || SHELL="/bin/bash"
export SHELL

"$DET_PYTHON_EXECUTABLE" -m pip install -q --user /opt/determined/wheels/determined*.whl

"$DET_PYTHON_EXECUTABLE" -m determined.exec.prep_container --resources

test -f "${STARTUP_HOOK}" && source "${STARTUP_HOOK}"

"$DET_PYTHON_EXECUTABLE" /run/determined/jupyter/check_idle.py &

READINESS_REGEX="^.*Jupyter Server .* is running.*$"
exec jupyter lab --ServerApp.port=${NOTEBOOK_PORT} \
                 --ServerApp.allow_origin="*" \
                 --ServerApp.base_url="/proxy/${DET_TASK_ID}/" \
                 --ServerApp.allow_root=True \
                 --ServerApp.ip="0.0.0.0" \
                 --ServerApp.open_browser=False \
                 --ServerApp.token="" \
                 --ServerApp.trust_xheaders=True \
    2> >(tee -p >("$DET_PYTHON_EXECUTABLE" /run/determined/check_ready_logs.py --ready-regex "${READINESS_REGEX}") >&2)

