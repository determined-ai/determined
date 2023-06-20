#!/usr/bin/env bash
set -e

# This is a workaround for CircleCI builds. Irrelevant if running on a local machine.
if [ -z $OPT_DEVBOX_PREFIX ]; then
    OPT_DEVBOX_PREFIX="$USER"
fi

cat <<EOF
bucket = "dev-instance-tf-state"
prefix = "$OPT_DEVBOX_PREFIX-slurmcluster"
EOF
