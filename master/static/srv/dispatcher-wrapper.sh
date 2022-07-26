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

# Fail on unexpected non-zero exit statuses.
set -e

# Controls debug logging for this method
DEBUG=0

# Clear all exported functions.  They are inherited into singularity containers
# since they are passed by environment variables.  One specific breaking example
# is the function that injects arguments into the which command.   These Redhat
# options are not supported on Ubuntu which breaks the which use in the entrypoints.
unset -f $(declare -Ffx | cut -f 3 -d ' ')

# When the task container is invoked via SLURM, we have
# to set the slot IDs from the Slurm-provided variable.
if [ "$DET_RESOURCES_TYPE" == "slurm-job" ]; then
    # One case for each device.Type in the Determined master source supported by slurm.
    case $DET_SLOT_TYPE in
        "cuda")
            export DET_SLOT_IDS="[${CUDA_VISIBLE_DEVICES}]"
            export DET_UNIQUE_PORT_OFFSET=$(echo $CUDA_VISIBLE_DEVICES | cut -d',' -f1)
            export DET_UNIQUE_PORT_OFFSET=${DET_UNIQUE_PORT_OFFSET:=0}

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
            else
                # If CUDA_VISIBLE_DEVICES is not set, then we default DET_SLOT_IDS the same that a
                # Determined agents deployment would, which should indicate to Determined to just use the
                # CPU.
                export DET_SLOT_IDS="[0]"
            fi
            ;;

        "rocm")
            export DET_SLOT_IDS="[${ROCR_VISIBLE_DEVICES}]"
            export DET_UNIQUE_PORT_OFFSET=$(echo $ROCR_VISIBLE_DEVICES | cut -d',' -f1)
            export DET_UNIQUE_PORT_OFFSET=${DET_UNIQUE_PORT_OFFSET:=0}

            if [ ! -z "$ROCR_VISIBLE_DEVICES" ]; then
                # Test if "rocm-smi" exists in the PATH before trying to invoking it.
                if [ ! type rocm-smi > /dev/null 2>&1 ]; then
                    echo "WARNING: rocm-smi not found.  May be unable to perform ROCM operations." 1>&2
                fi
            else
                # If ROCR_VISIBLE_DEVICES is not set, then we default DET_SLOT_IDS the same that a
                # Determined agents deployment would, which should indicate to Determined to just use the
                # CPU.
                export DET_SLOT_IDS="[0]"
            fi
            ;;

        "cpu")
            # For CPU only training, the "slot" we get is just the CPU, but it needs to be set.
            export DET_SLOT_IDS="[0]"
            export DET_UNIQUE_PORT_OFFSET=0
            ;;

        *)
            echo "ERROR: unsupported slot type: ${DET_SLOT_TYPE}"
            exit 1
            ;;
    esac
fi


# Debug log method
# Args: {Level} {Message}...
log() {
    if [ $DEBUG == 1 ]; then
       echo -e "$*" >&2
    fi 
}

# Container-local directory to host determined directory
# With --writable-tmpfs option / is writable by the user
# and private to the container instance.
LOCALTMP=/
# Source volume of all archives to be cloned
ROOT="/determined_local_fs"
# Base of the per-proc copy of tree
PROCDIR_ROOT="$ROOT/procs"
# Private copy of $ROOT for this $SLURM_PROCID
PROCDIR="$PROCDIR_ROOT/$SLURM_PROCID"

# Create clone of any directories under /dispatcher for this process and setup links
if [ -d $ROOT/run ] ; then
    mkdir -p $PROCDIR
    for dir in $ROOT/*; do
        if [[ -d $dir && $dir != $PROCDIR_ROOT ]] ; then
            log "INFO: Clone $dir -> $PROCDIR"
            cp -p -R $dir $PROCDIR >&2
        fi
    done

    if [ -d $LOCALTMP/determined ]; then
        log "ERROR: Container-private directory $LOCALTMP/determined already exists.\n$(ls -ld $LOCALTMP/determined)\nSingularity 3.7 or greater is required."
        log "INFO: ls -ld $LOCALTMP $(ls -ld $LOCALTMP)"
    fi

    # Container-local directory for links to container-specific /run/determined content
    log "INFO: Creating $LOCALTMP/determined"
    mkdir -m 0700 -p $LOCALTMP/determined >&2
    for dir in $ROOT/run/determined/*; do
        dirname=${dir##*/}
        log "INFO: ln -sfnT $PROCDIR/run/determined/${dirname} $LOCALTMP/determined/${dirname}"
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
if  [ "$DET_CONTAINER_LOCAL_TMP" == "1" ]; then
    # Create a per-container tmp
    mkdir -p $PROCDIR/tmp
    # Replace /tmp with a link to our private 
    rm -rf /tmp
    ln -fs $PROCDIR/tmp /
    log "DEBUG: Replaced tmp $(ls -l /tmp)"
fi


log "INFO: Resetting workdir to $DET_WORKDIR"
cd $DET_WORKDIR

log "INFO: executing $@" >&2
exec "$@"
