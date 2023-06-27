#!/bin/bash

CHANGES="$*"

if ! command -v golangci-lint >/dev/null; then
    echo "golangci-lint could not be found"
    exit
fi

CHANGED_DIRS=$(echo "$CHANGES" | xargs -n1 dirname | sort -u)

for DIR in $CHANGED_DIRS; do
    golangci-lint --build-tags integration run --timeout 10s "$DIR"
done

exit 0
