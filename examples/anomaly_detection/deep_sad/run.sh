#!/bin/bash

set -e

function det_e_create() {
  det e create "$@" | tee /dev/stderr \
    | grep -Eo 'Created experiment \d+' | awk -F ' ' '{print $3;}'
}

EXPERIMENT_ID=$(det_e_create const_ae.yaml .)
det e wait "$EXPERIMENT_ID"

CKPT_DIR=$(mktemp -d -t det-ckpt-XXXXXXXX)
det e download -o "$CKPT_DIR" "$EXPERIMENT_ID"
AE_STATE_FN=$(find "$CKPT_DIR" -name 'state_dict.pth' -print -quit)
cp "$AE_STATE_FN" ./ae_state_dict.pth

det e create const_main.yaml . -f

rm -r "$CKPT_DIR"
