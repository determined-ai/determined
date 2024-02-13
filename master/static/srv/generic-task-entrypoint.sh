#!/usr/bin/env bash

source /run/determined/task-setup.sh

set -e

STARTUP_HOOK="startup-hook.sh"
export PATH="/run/determined/pythonuserbase/bin:$PATH"

# If HOME is not explicitly set for a container, libcontainer (Docker) will
# try to guess it by reading /etc/password directly, which will not work with
# our libnss_determined plugin (or any user-defined NSS plugin in a container).
# The default is "/", but HOME must be a writable location for distributed
# training, so we try to query the user system for a valid HOME, or default to
# the working directory otherwise.
if [ "$HOME" = "/" ]; then
    HOME="$(
        set -o pipefail
        getent passwd "$(whoami)" | cut -d: -f6
    )" || HOME="$PWD"
    export HOME
fi

if [ -z "$DET_PYTHON_EXECUTABLE" ]; then
    export DET_PYTHON_EXECUTABLE="python3"
fi

# In order to be able to use a proxy when running a generic task, Python must be
# available in the container, and the "determined*.whl" must be installed,
# which contains the "determined/exec/prep_container.py" script that's needed
# to register the proxy with the Determined master.
"$DET_PYTHON_EXECUTABLE" -m determined.exec.prep_container --proxy --download_context_directory

set -x
test -f "${STARTUP_HOOK}" && source "${STARTUP_HOOK}"
set +x

if [ "$#" -eq 1 ]; then
    exec /bin/sh -c "$@"
else
    exec "$@"
fi
