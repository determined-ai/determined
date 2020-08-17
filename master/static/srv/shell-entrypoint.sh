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
# `man 8 sshd`.  Normal ssh mechanisms for overriding variables as part of the
# protocol (like TERM or LANG) will take precedence, as will normal mechanisms
# like a ~/.bashrc.  The purpose of this is to honor the environment variable
# settings as they are set for experiment or notebook configs, while still
# allowing customizations via normal ssh mechanisms.
#
# After openssh 8+ is the only version of openssh supported (that is, after we
# only support ubuntu >= 20.04), we can use the more obvious SetEnv option and
# skip this awkwardness.
vars="$(env | sed -e 's/=.*//')"
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
sed -i -e "s/^/$options /" "/run/determined/ssh/authorized_keys"

exec /usr/sbin/sshd "$@"
