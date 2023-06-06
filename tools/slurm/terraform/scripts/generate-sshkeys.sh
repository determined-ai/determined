#!/usr/bin/env bash
set -e

KEY_FILE=~/.slurmcluster/id_ed25519
if [ ! -f $KEY_FILE ]; then
    ssh-keygen -t ed25519 -N "" -C "$USER-slurmcluster" -f $KEY_FILE
fi
