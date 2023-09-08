#!/usr/bin/env bash

source /run/determined/task-signal-handling.sh
source /run/determined/task-logging-setup.sh

set -e

STARTUP_HOOK="startup-hook.sh"
export PATH="/run/determined/pythonuserbase/bin:$PATH"
if [ -z "$DET_PYTHON_EXECUTABLE" ]; then
    export DET_PYTHON_EXECUTABLE="python3"
fi

# Unlike trial and notebook entrypoints, the HOME directory does not need to be
# modified in this entrypoint because the HOME in the user's ssh session is set
# by sshd at a later time.

"$DET_PYTHON_EXECUTABLE" -m determined.exec.prep_container --resources --proxy

set -x
test -f "${STARTUP_HOOK}" && source "${STARTUP_HOOK}"
set +x

# Prepend each key in authorized_keys with a set of environment="KEY=VALUE"
# options to inject the entire docker environment into the eventual ssh
# session via an options in the authorized keys file.  See syntax described in
# `man 8 sshd`.  The purpose of this is to honor the environment variable
# settings as they are set for experiment or notebook configs, while still
# allowing customizations via normal ssh mechanisms.
#
# Not all variables should be overwritten this way; the HOME variable should be
# set by ssh, and the TERM, LANG, and LC_* variables should be passed in from
# the client.
#
# Normal mechanisms like a ~/.bashrc will override these variables.
#
# After openssh 8+ is the only version of openssh supported (that is, after we
# only support ubuntu >= 20.04), we can use the more obvious SetEnv option and
# skip this awkwardness.
#
# For HPC systems, bash module support uses variables that store functions
# of the form below (with embedded parenthesis or %% in the name).
#   BASH_FUNC_ml()=() {  eval $($LMOD_DIR/ml_cmd "$@")
#   BASH_FUNC_module%%=() {  eval `/opt/lib/modulecmd bash $*`
# so we also filter variables with parens or % in the name.

# extglob enables +() notation in patterns of ${parameter/pattern/string} notation
shopt -s extglob

# convert NUL-delimited environment key-value pairs in an array
# -d '' means "use NUL byte as delimiter"
# -t means "strip the delimiter"
mapfile -d '' -t kvps < <(env -0)

# iterate through each key-value pair in the array
options="$(
    for kvp in "${kvps[@]}"; do
        # Variable name is what comes before the first '='
        var="${kvp/=*/}"
        # Variable content starts after the first '='
        val="${kvp/#+([^=])=/}"

        # Filter names we shouldn't forward
        if [[ $var =~ ^(_|HOME|TERM|LANG|LC_.*)$ ]]; then
            continue
        fi

        # For slurm: filter variables with %% or () in the name
        if [[ $var =~ (%%|\(\)) ]]; then
            continue
        fi

        # Convert any explicit newline to \n
        val="${val//$'\n'/'\n'}"
        # Backslash-escape quotes so that sshd works.
        val="${val//\"/\\\"}"
        # Backslash-escape backslashes so that sed doesn't interpret them.
        val="${val//\\/\\\\}"
        # Backslash-escape forward slashes so that sed works.
        val="${val////\\/}"
        echo -n "environment=\"$var=$val\","
    done
)"

# In k8s, the files we inject into the container are injected via individual
# file-level bind mounts, which are effectively read-only in docker, so we are
# unable to edit authorized_keys in place.
unmodified="/run/determined/ssh/authorized_keys_unmodified"
modified="/run/determined/ssh/authorized_keys"
sed -e "s/^/$options /" "$unmodified" >"$modified"
# Ensure permissions are restrictive enough for ssh
chmod 600 "$modified"

READINESS_REGEX="Server listening on"

trap_and_forward_signals
/usr/sbin/sshd "$@" \
    2> >(tee -p >("$DET_PYTHON_EXECUTABLE" /run/determined/check_ready_logs.py --ready-regex "$READINESS_REGEX") >&2) &
wait_and_handle_signals $!
