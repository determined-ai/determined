#!/usr/bin/env bash

set -e

export PATH="/run/determined/pythonuserbase/bin:$PATH"

python3.6 -m pip install --user /opt/determined/wheels/determined*.whl

exec python3.6 -m determined.exec.gc_checkpoints "$@"
