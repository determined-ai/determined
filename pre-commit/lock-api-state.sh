#!/bin/sh

echo "TEST123"

git status --porcelain

# Regenerate proto/buf.image.bin
make get-deps-proto
make -C proto gen-buf-image

echo "TEST456"

git status --porcelain

# If proto/buf.image.bin has been modified locally, then we have changes to
# commit, and the status check returns a 1 and fails. Otherwise, it returns a 0
# and succeeds.
git diff --exit-code -- proto/buf.image.bin
exit $?
