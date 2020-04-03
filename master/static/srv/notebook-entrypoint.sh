#!/usr/bin/env bash

set -e

python3.6 -m pip install --user --upgrade --find-links /opt/determined/wheels determined determined-cli
jupyter lab --config /run/determined/workdir/jupyter-conf.py
