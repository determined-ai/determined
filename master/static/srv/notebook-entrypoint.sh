#!/usr/bin/env bash

python3.6 -m pip install --upgrade --find-links /opt/determined/wheels determined determined-cli
jupyter lab --config /run/determined/workdir/jupyter-conf.py
