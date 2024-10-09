#!/bin/sh

# Regenerate proto/buf.image.bin
make -C proto gen-buf-image

# If proto/buf.image.bin has been modified locally, then we have changes to
# commit, and the status check returns a 1 and fails. Otherwise, it returns a 0
# and succeeds.
git diff --exit-code -- proto/buf.image.bin
exit $?
