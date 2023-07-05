#!/usr/bin/env bash
set -e

export VMTIME=7200
export OPT_WORKLOAD_MANAGER="slurm"

while [[ $# -gt 0 ]]; do
    case $1 in
        -t)
            shift
            if [[ -n $1 && $1 != -* ]]; then
                export VMTIME=$1
                shift
            fi
            ;;
        -w)
            shift
            if [[ -n $1 && $1 != -* ]]; then
                export OPT_WORKLOAD_MANAGER=$1
                shift
            fi
            ;;
        *)
            echo "Invalid option: $1. Skipping..." >&2
            shift
            ;;
    esac
done

SSH_ALLOW_IP=$(curl -s https://checkip.amazonaws.com)
KEY_FILE=~/.slurmcluster/id_ed25519

# This is a workaround for CircleCI builds. Irrelevant if running on a local machine.
if [ -z $OPT_DEVBOX_PREFIX ]; then
    OPT_DEVBOX_PREFIX="$USER"
fi

if [[ $OPT_WORKLOAD_MANAGER == "slurm" ]]; then
    BOOT_DISK=$(grep "slurm" images.conf | cut -d':' -f2 | xargs)
elif [[ $OPT_WORKLOAD_MANAGER == "pbs" ]]; then
    BOOT_DISK=$(grep "pbs" images.conf | cut -d':' -f2 | xargs)
else
    echo >&2 "Invalid OPT_WORKLOAD_MANAGER value"
    exit 1
fi

cat <<EOF
name = "$OPT_DEVBOX_PREFIX-dev-box"
ssh_user = "$USER"
ssh_key_pub = "$KEY_FILE"
ssh_allow_ip = "$SSH_ALLOW_IP"
vmLifetimeSeconds = "$VMTIME"
workload_manager = "$OPT_WORKLOAD_MANAGER"
boot_disk = "projects/determined-ai/global/images/$BOOT_DISK"
EOF
