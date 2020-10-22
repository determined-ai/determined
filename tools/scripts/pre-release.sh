#!/bin/bash -ex

## Pre release hook. Run from the root of the project. Any arguments passed in are directly
## sent to `bumpversion`

PROJECT_ROOT=$(pwd)

if [[ $(git diff --shortstat 2> /dev/null | tail -n1) != "" ]]; then
    echo "git: there are dirty files."
    exit 1
elif [[ $(git status --porcelain 2>/dev/null| grep "^??") ]]; then
    echo "git: there are untracked files."
    exit 1
fi

## lock in current protobuf state
# Buf image binary
BUF_IMAGE=buf.image.bin
cd proto
make gen-buf-image
git add $BUF_IMAGE
git commit -m "lock backward buf compatibility check" || echo "buf image is already up to date"
cd $PROJECT_ROOT

## bump version
bumpversion $@
