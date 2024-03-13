#!/usr/bin/env bash

source /run/determined/task-setup.sh

set -e

# Use user's preferred SHELL in JupyterLab terminals.
SHELL="$(
    set -o pipefail
    getent passwd "$(whoami)" | cut -d: -f7
)" || SHELL="/bin/bash"
export SHELL

"$DET_PYTHON_EXECUTABLE" -m determined.exec.prep_container --resources --proxy --download_context_directory

STARTUP_HOOK="startup-hook.sh"
set -x
test -f "${STARTUP_HOOK}" && source "${STARTUP_HOOK}"
set +x

"$DET_PYTHON_EXECUTABLE" /run/determined/jupyter/check_idle.py &

JUPYTER_LAB_LOG_FORMAT="%(levelname)s: [%(name)s] %(message)s"
READINESS_REGEX='^.*Jupyter Server .* is running.*$'

jupyter lab --ServerApp.port=${NOTEBOOK_PORT} \
    --ServerApp.allow_origin="*" \
    --ServerApp.base_url="/proxy/${DET_TASK_ID}/" \
    --ServerApp.allow_root=True \
    --ServerApp.certfile=/run/determined/jupyter/jupyterCert.pem \
    --ServerApp.keyfile=/run/determined/jupyter/jupyterKey.key \
    --ServerApp.ip="0.0.0.0" \
    --ServerApp.open_browser=False \
    --ServerApp.token="" \
    --ServerApp.trust_xheaders=True \
    --Application.log_format="$JUPYTER_LAB_LOG_FORMAT" \
    --JupyterApp.log_format="$JUPYTER_LAB_LOG_FORMAT" \
    --ExtensionApp.log_format="$JUPYTER_LAB_LOG_FORMAT" \
    --LabServerApp.log_format="$JUPYTER_LAB_LOG_FORMAT" \
    --LabApp.log_format="$JUPYTER_LAB_LOG_FORMAT" \
    --ServerApp.log_format="$JUPYTER_LAB_LOG_FORMAT" \
    --ServerApp.root_dir="/" \
    --ContentsManager.preferred_dir="$PWD" \
    2> >(tee -p >("$DET_PYTHON_EXECUTABLE" /run/determined/check_ready_logs.py --ready-regex "${READINESS_REGEX}") >&2)
