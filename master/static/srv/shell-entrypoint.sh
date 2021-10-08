#!/usr/bin/env bash

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
    # don't need to keep any more old ones around.
    exec > >(multilog n2 "$STDOUT_FILE-rotate")  2> >(multilog n2 "$STDERR_FILE-rotate")
fi

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

"$DET_PYTHON_EXECUTABLE" -m pip install -q --user /opt/determined/wheels/determined*.whl

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
exec /usr/sbin/sshd "$@" \
    > >(tee -p >(exec "$DET_PYTHON_EXECUTABLE" /run/determined/check_ready_logs.py --ready-regex "$READINESS_REGEX")) \
    2> >(tee -p >(exec "$DET_PYTHON_EXECUTABLE" /run/determined/check_ready_logs.py --ready-regex "$READINESS_REGEX") >&2)
