#!/usr/bin/env bash
set -e

# This is a workaround for CircleCI builds. Irrelevant if running on a local machine.
if [ -z $OPT_DEVBOX_PREFIX ]; then
    OPT_DEVBOX_PREFIX="$USER"
fi

KEY_FILE=~/.slurmcluster/id_ed25519
if [ ! -f $KEY_FILE ]; then
    ssh-keygen -t ed25519 -N "" -C "$OPT_DEVBOX_PREFIX-slurmcluster" -f $KEY_FILE
fi
