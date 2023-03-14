#!/usr/bin/env bash

# Fail on unexpected non-zero exit statuses, but enable ERR trap.
set -eE
trap 'echo >&2 "FATAL: Unexpected error terminated dispatcher-wrapper container initialization.  See error messages above."' ERR

# Controls debug logging for this method
DEBUG=0

# TODO(DET-9074): Turn this only iff DET_DEBUG == true.
set -x

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

log_debug "DEBUG: Will utilize slots DET_SLOT_IDS $DET_SLOT_IDS"

log "INFO: executing $@" >&2
exec "$@"
