#!/usr/bin/env bash
set -e

SSH_ALLOW_IP=$(curl -s https://checkip.amazonaws.com)
KEY_FILE=~/.slurmcluster/id_ed25519

cat <<EOF
name = "$USER-dev-box"
ssh_user = "$USER"
ssh_key_pub = "$KEY_FILE"
ssh_allow_ip = "$SSH_ALLOW_IP"
EOF
