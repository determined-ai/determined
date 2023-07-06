#!/bin/bash

if ! command -v golangci-lint >/dev/null; then
    echo "golangci-lint could not be found (try ./master/get-deps.sh)" >&2
    exit 1
fi
set -xeo pipefail

CHANGED_DIRS=$(xargs -n1 dirname <<<"$*" | sort -u)

dir-filter() {
    (grep "^$1/" <<<"$2" | sed -e "s|^$1/||g") || true
}

golint() {
    dirs=$(dir-filter "$1" "$CHANGED_DIRS")
    if [ -z "$dirs" ]; then
        echo "No files to lint in $1, skipping."
        return
    fi
    pushd $1
    golangci-lint --build-tags integration run --timeout 30s $dirs
    popd
}

golint master
golint agent
