#!/usr/bin/env bash

set -e

python3.6 -m pip install --user /opt/determined/wheels/determined*.whl

jupyter lab --config /run/determined/workdir/jupyter-conf.py
