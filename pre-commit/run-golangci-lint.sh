#!/bin/bash

PRECOMMIT_GOLANGCI_LINT_NOT_FOUND_EXIT_CODE=${PRECOMMIT_GOLANGCI_LINT_NOT_FOUND_EXIT_CODE:-1}
if ! command -v golangci-lint >/dev/null; then
    echo "golangci-lint could not be found (try ./master/get-deps.sh)" >&2
    exit $PRECOMMIT_GOLANGCI_LINT_NOT_FOUND_EXIT_CODE
fi
set -xeo pipefail

CHANGED_DIRS=$(xargs -n1 dirname <<<"$*" | sort -u)

golint() {
    dirs=$(grep "^$1/" <<<"$CHANGED_DIRS" | sed -e "s|^$1/||g")
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
