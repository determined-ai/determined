#!/bin/sh

CONFIG_FILE="$1"

if ! command -v golangci-lint /dev/null 2>&1; then
    echo "golangci-lint could not be found"
    exit
fi

ERRS=$(golangci-lint run --new-from-rev="$(git rev-parse HEAD)" "$CONFIG_FILE")

if [ -n "${ERRS}" ]; then
    echo "${ERRS}"
    exit 1
fi
exit 0
