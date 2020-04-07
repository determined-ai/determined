#!/usr/bin/env bash

python3.6 -m pip install --upgrade --find-links /opt/determined/wheels determined
exec python3.6 -m determined.exec.gc_checkpoints "$@"
