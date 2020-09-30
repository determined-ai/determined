#!/bin/bash

set -e
set -x

# Create symbolic links from well-known files to this process's STDOUT and
# STDERR. Anything written to those files will be inserted into the output
# streams of this process, allowing distributed training logs to route through
# individual containers rather than all going through SSH back to agent 0.
ln -sf /proc/$$/fd/1 /run/determined/train/stdout.log
ln -sf /proc/$$/fd/2 /run/determined/train/stderr.log

WORKING_DIR="/run/determined/workdir"
STARTUP_HOOK="startup-hook.sh"
export PATH="/run/determined/pythonuserbase/bin:$PATH"

# If HOME is not explicitly set for a container, libcontainer (Docker) will
# try to guess it by reading /etc/password directly, which will not work with
# our linss_determined plugin (or any user-defined NSS plugin in a container).
# The default is "/", but HOME must be a writable location for distributed
# training, so we try to query the user system for a valid HOME, or default to
# the working directory otherwise.
if [ "$HOME" = "/" ] ; then
    HOME="$(set -o pipefail; getent passwd "$(whoami)" | cut -d: -f6)" || HOME="$WORKING_DIR"
    export HOME
fi

python3.6 -m pip install -q --user /opt/determined/wheels/determined*.whl

cd ${WORKING_DIR} && test -f "${STARTUP_HOOK}" && source "${STARTUP_HOOK}"
exec python3.6 -m determined.exec.harness "$@"
