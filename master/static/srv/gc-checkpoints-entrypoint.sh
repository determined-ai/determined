#!/usr/bin/env bash

set -e

python3.6 -m pip install --user --upgrade --find-links /opt/determined/wheels determined
exec python3.6 -m determined.exec.gc_checkpoints "$@"
