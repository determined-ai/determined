#!/bin/bash -ex

if [ "$#" -gt 1 ] || [ -n "$1" -a "$1" != "--release" ]; then
    echo "usage: $0 [--release]" >&2
    exit 1
fi

# check for dirty changes
if [[ -n "$(git status --porcelain)" ]]; then
    echo "untracked or dirty files are not allowed, cleanup before running release-party-bumpenvs.sh"
    exit 1
fi

if [ "$1" == "--release" ]; then
    python tools/scripts/retag-bumpenvs-yaml.py tools/scripts/bumpenvs.yaml \
    "$(cat tools/scripts/environments-target.txt)" "$(cat VERSION)" --release
else
    python tools/scripts/retag-bumpenvs-yaml.py tools/scripts/bumpenvs.yaml \
    "$(cat tools/scripts/environments-target.txt)" "$(cat VERSION)"
fi

python tools/scripts/bumpenvs.py tools/scripts/bumpenvs.yaml


git add --update
git commit -m "chore: bump environment image tags"
