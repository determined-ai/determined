#!/bin/bash -ex

## lock in current protobuf state

# make gen-buf-iamge ensures that it starts with a clean git state
make -C proto gen-buf-image
if [[ -z "$(git status --porcelain)" ]]; then
    echo "buf image is already up to date"
    exit 0
fi
git add --all
git commit -m "chore: lock api state for backward compatibility check"
