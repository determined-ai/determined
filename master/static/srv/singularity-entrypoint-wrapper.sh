#!/usr/bin/env bash

# Fail on unexpected non-zero exit statuses, but enable ERR trap.
set -eE
trap 'echo >&2 "FATAL: Unexpected error terminated dispatcher-wrapper container initialization.  See error messages above."' ERR

# Controls debug logging for this method
DEBUG=0

if [ $DET_DEBUG = "1" ]; then
    set -x
fi

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

for encoded_env_var_name in $(echo $DET_B64_ENCODED_ENVVARS | tr "," "\n"); do
    decoded_env_var=$(echo ${!encoded_env_var_name} | base64 --decode)
    export ${encoded_env_var_name}="$decoded_env_var"
done

# Source volume of all archives to be cloned
ROOT="/determined-local-fs"
# Base of the per-proc copy of tree
PROCDIR_ROOT="$ROOT/procs"
# Private copy of $ROOT for this $DET_CONTAINER_ID
PROCDIR="$PROCDIR_ROOT/$DET_CONTAINER_ID"

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

log_debug "  ROCR_VISIBLE_DEVICES: $ROCR_VISIBLE_DEVICES"
log_debug "  CUDA_VISIBLE_DEVICES: $CUDA_VISIBLE_DEVICES"
log_debug "NVIDIA_VISIBLE_DEVICES: $NVIDIA_VISIBLE_DEVICES"
log_debug "          DET_SLOT_IDS: $DET_SLOT_IDS"

log "INFO: executing $@" >&2
exec "$@"
