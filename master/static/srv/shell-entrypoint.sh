#!/usr/bin/env bash

source /run/determined/task-signal-handling.sh
source /run/determined/task-logging-setup.sh
trap 'source /run/determined/task-logging-teardown.sh' EXIT

set -e

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

# Unlike trial and notebook entrypoints, the HOME directory does not need to be
# modified in this entrypoint because the HOME in the user's ssh session is set
# by sshd at a later time.

if [ -z "$DET_SKIP_PIP_INSTALL" ]; then
	"$DET_PYTHON_EXECUTABLE" -m pip install -q --user /opt/determined/wheels/determined*.whl
fi

"$DET_PYTHON_EXECUTABLE" -m determined.exec.prep_container --resources

test -f "${STARTUP_HOOK}" && source "${STARTUP_HOOK}"

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
blacklist="^(_|HOME|TERM|LANG|LC_.*)"
vars="$(env | sed -E -e "s/=.*//; /$blacklist/d")"
options="$(
    for var in $vars ; do
        # Note that the syntax ${!var} is for a double dereference.
        val="${!var}"
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
sed -e "s/^/$options /" "$unmodified" > "$modified"

READINESS_REGEX="Server listening on"

trap_and_forward_signals
/usr/sbin/sshd "$@" \
    2> >(tee -p >("$DET_PYTHON_EXECUTABLE" /run/determined/check_ready_logs.py --ready-regex "$READINESS_REGEX") >&2) &
wait_and_handle_signals $!
