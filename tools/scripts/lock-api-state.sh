#!/bin/bash -ex

# Buf image binary
BUF_IMAGE=buf.image.bin
PROJECT_ROOT=$(pwd)

## lock in current protobuf state
cd proto
make gen-buf-image
git add $BUF_IMAGE
git commit -m "chore: lock api state for backward compatibility check" || echo "buf image is already up to date"
cd $PROJECT_ROOT
