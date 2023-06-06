#!/usr/bin/env bash
set -e

cat <<EOF
bucket = "dev-instance-tf-state"
prefix = "$USER-slurmcluster"
EOF
