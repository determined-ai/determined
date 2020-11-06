#!/bin/bash -ex

## lock in current protobuf state

# make gen-buf-image ensures that it starts with a clean git state
make -C proto gen-buf-image
# check to see if gen-buf-image resulted in any file changes or not
if [[ -z "$(git status --porcelain)" ]]; then
    echo "buf image is already up to date"
    exit 0
fi
git add --update
git commit -m "chore: lock api state for backward compatibility check"
