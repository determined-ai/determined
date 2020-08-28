#!/usr/bin/env bash

set -e

export PATH="/run/determined/pythonuserbase/bin:$PATH"

# Unlike trial and notebook entrypoints, the HOME directory does not need to be
# modified in this entrypoint because the HOME in the user's ssh session is set
# by sshd at a later time.

python3.6 -m pip install --user /opt/determined/wheels/determined*.whl

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

exec /usr/sbin/sshd "$@"
