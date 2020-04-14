#!/usr/bin/env bash

set -e

python3.6 -m pip install --user /opt/determined/wheels/determined*.whl

exec python3.6 -m determined.exec.gc_checkpoints "$@"
