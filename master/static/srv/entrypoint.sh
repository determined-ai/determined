#!/bin/bash

# TODO(brad); BEGIN: Pull into task-logging-setup.sh
STDOUT_FILE=/run/determined/train/logs/stdout.log
STDERR_FILE=/run/determined/train/logs/stderr.log

mkdir -p "$(dirname "$STDOUT_FILE")" "$(dirname "$STDERR_FILE")"

# Create symbolic links from well-known files to this process's STDOUT and
# STDERR. Anything written to those files will be inserted into the output
# streams of this process, allowing distributed training logs to route through
# individual containers rather than all going through SSH back to agent 0.
ln -sf /proc/$$/fd/1 "$STDOUT_FILE"
ln -sf /proc/$$/fd/2 "$STDERR_FILE"

if [ -n "$DET_K8S_LOG_TO_FILE" ]; then
    # To do logging with a sidecar in Kubernetes, we need to log to files that
    # can then be read from the sidecar. To avoid a disk explosion, we need to
    # layer on some rotation. multilog is a tool that automatically writes its
    # stdin to rotated log files; the following line pipes stdout and stderr of
    # this process to separate multilog invocations. "n2" means to only store
    # one old log file -- the logs are being streamed out by Fluent Bit, so we
    # don't need to keep any more old ones around. Create the dirs ahead of time
    # so they are 0755 (when they don't exist, multilog makes them 0700 and
    # Fluent Bit can't access them with the non-root user).
    STDOUT_ROTATE_DIR="$STDOUT_FILE-rotate"
    STDERR_ROTATE_DIR="$STDERR_FILE-rotate"
    mkdir -p -m 755 $STDOUT_ROTATE_DIR
    mkdir -p -m 755 $STDERR_ROTATE_DIR
    exec > >(multilog n2 "$STDOUT_ROTATE_DIR")  2> >(multilog n2 "$STDERR_ROTATE_DIR")
fi
# TODO(brad); END: Pull into task-logging-setup.sh

set -e
set -x

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

"$DET_PYTHON_EXECUTABLE" -m pip install -q --user /opt/determined/wheels/determined*.whl

"$DET_PYTHON_EXECUTABLE" -m determined.exec.prep_container --trial --resources

test -f "${STARTUP_HOOK}" && source "${STARTUP_HOOK}"

# Do rendezvous last, to ensure all launch layers start around the same time.
"$DET_PYTHON_EXECUTABLE" -m determined.exec.prep_container --rendezvous

exec "$DET_PYTHON_EXECUTABLE" -m determined.exec.launch_autohorovod "$@"
