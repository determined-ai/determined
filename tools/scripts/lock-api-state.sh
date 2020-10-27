#!/bin/bash -ex

# Buf image binary
BUF_IMAGE=buf.image.bin
PROJECT_ROOT=$(pwd)

if [[ $(git diff --shortstat 2> /dev/null | tail -n1) != "" ]]; then
    echo "git: there are dirty files."
    exit 1
elif [[ $(git status --porcelain 2>/dev/null| grep "^??") ]]; then
    echo "git: there are untracked files."
    exit 1
fi

## lock in current protobuf state
cd proto
make check
make gen-buf-image
git add $BUF_IMAGE
git commit -m "chore: lock api state for backward compatibility check" || echo "buf image is already up to date"
cd $PROJECT_ROOT
