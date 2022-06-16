#!/bin/bash

# When the task container is invoked via SLURM, we have
# to set the slot IDs.
if [ "$DET_RESOURCES_TYPE" == "slurm-job" ]; then
    export DET_SLOT_IDS="[${CUDA_VISIBLE_DEVICES}]"

    if [ ! -z "$CUDA_VISIBLE_DEVICES" ]; then
        # Test if "nvidia-smi" exists in the PATH before trying to invoking it.
        if type nvidia-smi > /dev/null 2>&1 ; then
            # For Nvidia GPUS, the slot IDs are the device index. Replace the
            # newline characters with commas and enclose in square brackets.
            # But only include GPUS that are in the CUDA_VISIBLE_DEVICES=0,1,...
            VISIBLE_SLOTS="$(nvidia-smi --query-gpu=index --format=csv,noheader | sed -z 's/\n/,/g;s/,$/\n/')"
            for device in ${CUDA_VISIBLE_DEVICES//,/ } ; do 
                if [[ ! "$VISIBLE_SLOTS" == *"$device"* ]]; then
                    echo "WARNING: nvidia-smi reports visible CUDA devices as ${VISIBLE_SLOTS} but does not contain ${device}.  May be unable to perform CUDA operations." 1>&2
                fi 
            done
 
        else
            echo "WARNING: nvidia-smi not found.  May be unable to perform CUDA operations." 1>&2
        fi
    fi
fi

source /run/determined/task-signal-handling.sh
source /run/determined/task-logging-setup.sh

set -e

STARTUP_HOOK="startup-hook.sh"
export PATH="/run/determined/pythonuserbase/bin:$PATH"
if [ -z "$DET_PYTHON_EXECUTABLE" ] ; then
    export DET_PYTHON_EXECUTABLE="python3"
fi
if ! which "$DET_PYTHON_EXECUTABLE" >/dev/null 2>&1 ; then
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

if [ -z "$DET_SKIP_PIP_INSTALL" ]; then
    "$DET_PYTHON_EXECUTABLE" -m pip install -q --user /opt/determined/wheels/determined*.whl
fi

"$DET_PYTHON_EXECUTABLE" -m determined.exec.prep_container --trial --resources

test -f "${STARTUP_HOOK}" && source "${STARTUP_HOOK}"

# Do rendezvous last, to ensure all launch layers start around the same time.
"$DET_PYTHON_EXECUTABLE" -m determined.exec.prep_container --rendezvous

trap_and_forward_signals
"$DET_PYTHON_EXECUTABLE" -m determined.exec.launch "$@" &
wait_and_handle_signals $!
