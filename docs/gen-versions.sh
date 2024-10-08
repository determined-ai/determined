#!/bin/bash

# Script to return all versions on a branch between a given EPOCH and HEAD,
# while removing specific excluded versions. This is used to generate the
# versions.json file the Sphinx version picker uses. This also replaces some
# functionality that used to be present in gen-versions.py, using GitPython,
# which returned version tags in a way that doesn't match how we're working with
# version tags elsewhere. I.e. this git command is sensitive to where HEAD is in
# the DAG, and will not return versions that could logically come after where
# HEAD is; previous versions would return all later tags regardless.

EPOCH="0.21.0"

VERSIONS=$(git \
    -c versionsort.suffix='-rc' \
    tag \
    --sort='v:refname:short' \
    --format='%(refname:short)' \
    --no-contains=$(git merge-base HEAD main) \
    --contains=$(git rev-parse $(git merge-base HEAD ${EPOCH})~1) \
    | grep -E -v 'v0.12|-ee' \
    | grep -E -o '\d+\.\d+\.\d+$')

comm -2 -3 <(cat <<<"${VERSIONS}") exclude-versions.txt | sort -Vr
