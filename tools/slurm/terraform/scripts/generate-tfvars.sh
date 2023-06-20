#!/usr/bin/env bash
set -e

SSH_ALLOW_IP=$(curl -s https://checkip.amazonaws.com)
KEY_FILE=~/.slurmcluster/id_ed25519

# This is a workaround for CircleCI builds. Irrelevant if running on a local machine.
if [ -z $OPT_DEVBOX_PREFIX ]; then
    OPT_DEVBOX_PREFIX="$USER"
fi

cat <<EOF
name = "$OPT_DEVBOX_PREFIX-dev-box"
ssh_user = "$USER"
ssh_key_pub = "$KEY_FILE"
ssh_allow_ip = "$SSH_ALLOW_IP"
EOF
