#!/usr/bin/env bash
# Usage:
# dispather-entrypoint.sh: {realEntryPointArgs}...
#
# This is the wrapper script added around the intended determined
# entrypoint script to provide dispatcher-specific initialization
# for singularity.  In particular, it processes the /determined_local_fs volume
# and clones it under /determined_local_fs/procs/# for this particular process ($SLURM_PROCID).
# It then adds softlinks for each subdirectory to re-direct it
# (via $LOCALTMP/determined/xx) -> /determined_local_fs/procs/#/run/determined/xx
#
# The links from /run/determined are provided by the DAI master dispatcher RM
# via softlinks in the archives provided to the dispatcher and expanded in-place,
# so this script only needs to handle the cloning of the file system per process
# and setting up links from $LOCALTMP/determined/xx to the local copy of
# in the /determined_local_fs/procs/# tree.
#
# This is additionally a place for all common behavior specific to SLURM/Singularity
# which addresses:
#    - DET_SLOT_IDS inherited from SLURM-provided CUDA_VISIBLE_DEVICES/ROCR_VISIBLE_DEVICES
#    - DET_UNIQUE_PORT_OFFSET inherited from SLURM-provided least(CUDA_VISIBLE_DEVICES/ROCR_VISIBLE_DEVICES)

# Fail on unexpected non-zero exit statuses, but enable ERR trap.
set -eE
trap 'echo >&2 "FATAL: Unexpected error terminated dispatcher-wrapper container initialization.  See error messages above."' ERR

# Controls debug logging for this method
DEBUG=0

# Clear all exported functions.  They are inherited into singularity containers
# since they are passed by environment variables.  One specific breaking example
# is the function that injects arguments into the which command.  These Red Hat
# options are not supported on Ubuntu which breaks the which use in the entrypoints.
unset -f $(declare -Ffx | cut -f 3 -d ' ')

# Debug log method (logged only if DEBUG=1)
# Args: {Level} {Message}...
log_debug() {
    if [ $DEBUG == 1 ]; then
        echo -e "$*" >&2
    fi
}

# Unconditional log method
# Args: {Level} {Message}...
log() {
    echo -e "$*" >&2
}

# CUDA_VISIBLE_DEVICES is set by both SLURM and PBS. However, there can be differences in the
# format used by each of them. SLURM is guaranteed to set the value using simple number like
# "0,1,2" whereas PBS can sometimes set the value using GPU UUID like
# "GPU-UUID1,GPU-UUID2,GPU-UUID3".
# We follow the format used by SLURM. So, before using the value of CUDA_VISIBLE_DEVICES we have to
# make sure it is made of simple numbers instead of UUIDs. If it is not, then we update it to match
# that format.
# convert_to_gpu_numbers will inspect the value of CUDA_VISIBLE_DEVICES and convert it to gpu
# numbers format. If the input is already in the simple number format the same value is returned.
# If the input based on GPU UUID format, it it converted to simple number format and returned as a
# string. If there is an error during the conversion, the error is logged and the function returns
# the existing CUDA_VISIBLE_DEVICES value.
# TODO: Need to handle Multi Instance GPU (MIG) Format.
# Refer: https://docs.nvidia.com/datacenter/tesla/pdf/NVIDIA_MIG_User_Guide.pdf Section 9.6.1 for
# further information on MIG format
convert_to_gpu_numbers() {
    # Process the value of CUDA_VISIBLE_DEVICES and store the values in an array.
    # IFS flag is set to "," to process the string as a comma separated list.
    IFS=',' read -r -a cuda_device_ids <<<"$CUDA_VISIBLE_DEVICES"
    # Check if the first element is a number.
    if [[ ${cuda_device_ids[0]} =~ ^[[:digit:]]+$ ]]; then
        # Return the value as it is, if it is already in simple number format.
        echo "${CUDA_VISIBLE_DEVICES}"
    else
        # Update the value of CUDA_VISIBLE_DEVICES.
        cuda_devices_string=""
        error_flag=0
        # Below for loop will creates a string in the format "0,1,2,"
        for gpu_id in "${cuda_device_ids[@]}"; do
            # Retrieve gpu id in the simple number format using the nvidia-smi command.
            simple_gpu_id=$(nvidia-smi --query-gpu=index --format=csv,noheader -i ${gpu_id})
            # If the command failed log warning and return the existing value as it is.
            if [ $? -ne 0 ]; then
                log "ERROR: Failed to retrieve index for GID ${gpu_id} using nvidia-smi." 1>&2
                error_flag=1
                break
            fi
            cuda_devices_string+="$simple_gpu_id"
            cuda_devices_string+=","
        done
        if [[ error_flag -ne 0 ]]; then
            # Return the value as it is in case of an error.
            echo "${CUDA_VISIBLE_DEVICES}"
        else
            # Return the number format string excluding the trailing comma.
            echo "${cuda_devices_string::-1}"
        fi
    fi
}

# Set DET_CONTAINER_ID as the SLURM_PROCID. Usually it would be a Docker container ID.
export DET_CONTAINER_ID="$SLURM_PROCID"

# Container-local directory to host determined directory and links (default to /)
LOCALTMP=${DET_LOCALTMP:-/}
# Source volume of all archives to be cloned
ROOT="/determined_local_fs"
# Base of the per-proc copy of tree
PROCDIR_ROOT="$ROOT/procs"
# Private copy of $ROOT for this $SLURM_PROCID
PROCDIR="$PROCDIR_ROOT/$SLURM_PROCID"

# Create clone of any directories under /dispatcher for this process and setup links
if [ -d $ROOT/run ]; then
    mkdir -p $PROCDIR
    for dir in $ROOT/*; do
        if [[ -d $dir && $dir != $PROCDIR_ROOT ]]; then
            log_debug "INFO: Clone $dir -> $PROCDIR"
            cp -p -R $dir $PROCDIR >&2
        fi
    done

    if [ -d $LOCALTMP/determined ]; then
        log "ERROR: Container-private directory $LOCALTMP/determined already exists.\n$(ls -ld $LOCALTMP/determined)\nSingularity 3.7 or greater is required."
        log "INFO: ls -ld $LOCALTMP $(ls -ld $LOCALTMP)"
    fi

    # Container-local directory for links to container-specific /run/determined content
    log_debug "INFO: Creating $LOCALTMP/determined"
    mkdir -m 0700 -p $LOCALTMP/determined >&2
    for dir in $ROOT/run/determined/*; do
        dirname=${dir##*/}
        log_debug "DEBUG: ln -sfnT $PROCDIR/run/determined/${dirname} $LOCALTMP/determined/${dirname}"
        if [ ! -w $PROCDIR/run/determined ]; then
            log "ERROR: User$(id) does not have write access to $PROCDIR/run/determined/${dirname}.  You may have may not have properly configured your determined agent user/group."
        fi
        if [ ! -w $LOCALTMP/determined ]; then
            log "ERROR: User $(id) does not have write access to $LOCALTMP/determined/${dirname}. You may have may not have properly configured your determined agent user/group."
        fi
        ln -sfnT $PROCDIR/run/determined/${dirname} $LOCALTMP/determined/${dirname} >&2
    done
fi

# Localize /tmp as a private folder in the container, if requested.
if [ "$DET_CONTAINER_LOCAL_TMP" == "1" ]; then
    # Create a per-container tmp
    mkdir -p $PROCDIR/tmp
    # Replace /tmp with a link to our private
    if rmdir /tmp; then
        ln -fs $PROCDIR/tmp /
        log_debug "DEBUG: Replaced tmp $(ls -l /tmp)"
    else
        log "ERROR: Unable to replace /tmp with per-container $PROCDIR/tmp (potential bind mount conflict?).  Free space in /tmp may be limited."
    fi
fi

# When the task container is invoked via SLURM, we have
# to set the DET slot IDs from the Slurm-provided variable.
if [ "$DET_RESOURCES_TYPE" == "slurm-job" ]; then
    # One case for each device.Type in the Determined master source supported by slurm.
    case $DET_SLOT_TYPE in
        "cuda")
            if [ ! -z "$CUDA_VISIBLE_DEVICES" ]; then
                # Ensure CUDA_VISIBLE_DEVICES is in the format of simple numbers like "0,1,2"
                export CUDA_VISIBLE_DEVICES="$(convert_to_gpu_numbers)"
            fi

            export DET_UNIQUE_PORT_OFFSET=${DET_UNIQUE_PORT_OFFSET:=0}

            if [ ! -z "$CUDA_VISIBLE_DEVICES" ]; then
                export DET_SLOT_IDS="[${CUDA_VISIBLE_DEVICES}]"
                export DET_UNIQUE_PORT_OFFSET=$(echo $CUDA_VISIBLE_DEVICES | cut -d',' -f1)

                # Test if "nvidia-smi" exists in the PATH before trying to invoking it.
                if type nvidia-smi >/dev/null 2>&1; then
                    # For Nvidia GPUS, the slot IDs are the device index. Replace the
                    # newline characters with commas and enclose in square brackets.
                    # But only include GPUS that are in the CUDA_VISIBLE_DEVICES=0,1,...
                    VISIBLE_SLOTS="$(nvidia-smi --query-gpu=index --format=csv,noheader | sed -z 's/\n/,/g;s/,$/\n/')"
                    for device in ${CUDA_VISIBLE_DEVICES//,/ }; do
                        if [[ $VISIBLE_SLOTS != *"$device"* ]]; then
                            log "WARNING: nvidia-smi reports visible CUDA devices as ${VISIBLE_SLOTS} but does not contain ${device}.  May be unable to perform CUDA operations." 1>&2
                        fi
                    done
                else
                    log "WARNING: nvidia-smi not found.  May be unable to perform CUDA operations." 1>&2
                fi
            elif [ -z "$DET_SLOT_IDS" ]; then
                # If CUDA_VISIBLE_DEVICES and DET_SLOT_IDS are not set, then we default DET_SLOT_IDS the same as
                # Determined agents deployment would, which should indicate to Determined to just use one
                # CPU.  If slots_per_node is specified, DET_SLOT_IDS will be provided indicating the slots to use.
                export DET_SLOT_IDS="[0]"
            fi
            ;;

        "rocm")
            export DET_UNIQUE_PORT_OFFSET=${DET_UNIQUE_PORT_OFFSET:=0}

            # ROCm command rocm-smi is implemented as a python script.  With singularity --rocm
            # the script from the host is mapped into the container.   If the host is a Redhat variant
            # the python interpreter is referenced as /usr/libexec/platform-python  which is not available
            # inside the Determined Unbuntu-based enviornment container, and therefore the command fails. This code
            # detects this situation, creates a wrapper script for rocm-smi on the path that supplies the python3
            # interpereter from within the container.   The script is created in /run/determined/pythonuserbase/bin
            # which is added as the first element in the path in all the entrypoints scripts.
            if [[ -x /usr/bin/rocm-smi ]]; then
                if grep -s /usr/libexec/platform-python /usr/bin/rocm-smi; then
                    mkdir -p /run/determined/pythonuserbase/bin/
                    echo -e '#!/bin/bash\npython3 /usr/bin/rocm-smi $*' >/run/determined/pythonuserbase/bin/rocm-smi
                    chmod +x /run/determined/pythonuserbase/bin/rocm-smi
                    log "INFO: Adding rocm-smi wrapper script /run/determined/pythonuserbase/bin/rocm-smi." 1>&2
                fi
            fi

            if [ ! -z "$ROCR_VISIBLE_DEVICES" ]; then
                export DET_SLOT_IDS="[${ROCR_VISIBLE_DEVICES}]"
                export DET_UNIQUE_PORT_OFFSET=$(echo $ROCR_VISIBLE_DEVICES | cut -d',' -f1)

                # Test if "rocm-smi" exists in the PATH before trying to invoking it.
                if [ ! type rocm-smi ] >/dev/null 2>&1; then
                    log "WARNING: rocm-smi not found.  May be unable to perform ROCM operations." 1>&2
                fi
            elif [ -z "$DET_SLOT_IDS" ]; then
                # If ROCR_VISIBLE_DEVICES is not set, then we default DET_SLOT_IDS the same as
                # Determined agents deployment would, which should indicate to Determined to just use the
                # CPU.
                export DET_SLOT_IDS="[0]"
            fi
            ;;

        "cpu")
            if [ -z "$DET_SLOT_IDS" ]; then
                # For CPU only training, the "slot" we get is just the CPU, but it needs to be set.
                export DET_SLOT_IDS="[0]"
            fi
            export DET_UNIQUE_PORT_OFFSET=0
            ;;

        *)
            log "ERROR: unsupported slot type: ${DET_SLOT_TYPE}"
            exit 1
            ;;
    esac
fi

# The default "max locked memory" is 64, which is too low for IB
# If we are getting the default value in the container, then raise it to
# unlimited.   If it is set to some other value, leave it as-is to
# allow a customer override.
if [ $(ulimit -l) == 64 ]; then
    log_debug "DEBUG: Setting (max locked memory) ulimit -l unlimited"
    ulimit -l unlimited
fi

# When running under podman rootless, the default user on entry is root/uid=0 inside the
# container, which maps to the launching user outside the container.  In this case,
# rewrite the determined agent user uid/gid in /run/determined/etc/passwd to be
# 0:0 to match.
#
# cat  /run/determined/etc/passwd
#     username:x:1001:39::/run/determined/workdir:/bin/bash
#
# This maps the launching user into the same account when entering vis SSH
# as the user what is executing the entry-point script here, and will
# result in:
# cat  /run/determined/etc/passwd
#     username:x:0:0::/run/determined/workdir:/bin/bash
if [ $(whoami) == "root" ] && [ -r /run/determined/etc/passwd ]; then
    log_debug "DEBUG: Running as root inside container, changing agent user passwd entry to uid/gid 0/0."
    sed -i "s/\([a-zA-Z0-9]\+\):x:[0-9]\+:[0-9]\+:/\1:x:0:0:/" /run/determined/etc/passwd
fi

log_debug "DEBUG: Will utilize slots DET_SLOT_IDS $DET_SLOT_IDS"

log "INFO: Setting workdir to $DET_WORKDIR"
cd $DET_WORKDIR

log "INFO: executing $@" >&2
exec "$@"
