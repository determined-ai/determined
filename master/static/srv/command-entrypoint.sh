#!/usr/bin/env bash

STDOUT_FILE=/run/determined/train/logs/stdout.log
STDERR_FILE=/run/determined/train/logs/stderr.log

mkdir -p "$(dirname "$STDOUT_FILE")" "$(dirname "$STDERR_FILE")"

# Explaination for this is found in ./entrypoint.sh.
ln -sf /proc/$$/fd/1 "$STDOUT_FILE"
ln -sf /proc/$$/fd/2 "$STDERR_FILE"
if [ -n "$DET_K8S_LOG_TO_FILE" ]; then
    exec > >(multilog n2 "$STDOUT_FILE-rotate")  2> >(multilog n2 "$STDERR_FILE-rotate")
fi

set -e

if [ "$#" -eq 1 ];
then
    exec /bin/sh -c "$@"
else
    exec "$@"
fi
