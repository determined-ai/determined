#!/usr/bin/env bash
# This script displays terraform variable assignments intended
# to be written to default.tfvars based upon the CLI arguments.
set -e

VMTIME=7200
OPT_WORKLOAD_MANAGER="slurm"
# No default machine type
MACHINE_TYPE=
# Type of GPU
GPU_TYPE=
# Number of GPUs
GPU_COUNT=

while [[ $# -gt 0 ]]; do
    case $1 in
        -t)
            shift
            if [[ -n $1 && $1 != -* ]]; then
                VMTIME=$1
                shift
            fi
            ;;
        -w)
            shift
            if [[ -n $1 && $1 != -* ]]; then
                OPT_WORKLOAD_MANAGER=$1
                shift
            fi
            ;;
        -c)
            # Handled by slurmcluster.sh
            shift 2
            ;;
        -m)
            shift
            if [[ -n $1 && $1 != -* ]]; then
                MACHINE_TYPE=$1
                shift
            fi
            ;;
        -g)
            shift
            if [[ -n $1 && $1 != -* ]]; then
                GPU_TYPE=$(echo $1 | cut -d : -f 1)
                GPU_COUNT=$(echo $1 | cut -d : -f 2)
                if [[ -z $GPU_COUNT ]]; then
                    echo "Bad option format -g {gcp_name}:{count}" >&2
                    echo "  Example: -g nvidia-tesla-t4:4" >&2
                fi
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
name              = "$OPT_DEVBOX_PREFIX-dev-box"
ssh_user          = "$USER"
ssh_key_pub       = "$KEY_FILE"
ssh_allow_ip      = "$SSH_ALLOW_IP"
vmLifetimeSeconds = "$VMTIME"
workload_manager  = "$OPT_WORKLOAD_MANAGER"
boot_disk         = "projects/determined-ai/global/images/$BOOT_DISK"
EOF
if [[ -n $MACHINE_TYPE ]]; then
    echo "machine_type      = \"$MACHINE_TYPE\""
fi
if [[ -n $GPU_TYPE ]]; then
    echo "gpus   = {"
    echo "   type: \"$GPU_TYPE\""
    echo "   count: $GPU_COUNT"
    echo "}"
    echo "allow_stopping_for_update        = true"
fi
