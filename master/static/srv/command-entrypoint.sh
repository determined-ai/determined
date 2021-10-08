#!/usr/bin/env bash

STDOUT_FILE=/run/determined/train/logs/stdout.log
STDERR_FILE=/run/determined/train/logs/stderr.log

mkdir -p "$(dirname "$STDOUT_FILE")" "$(dirname "$STDERR_FILE")"

# Create symbolic links from well-known files to this process's STDOUT and
# STDERR. Anything written to those files will be inserted into the output
# streams of this process, allowing distributed training logs to route through
# individual containers rather than all going through SSH back to agent 0.
ln -sf /proc/$$/fd/1 "$STDOUT_FILE"
ln -sf /proc/$$/fd/2 "$STDERR_FILE"

if [ -n "$DET_K8S_LOG_TO_FILE" ]; then
    # To do logging with a sidecar in Kubernetes, we need to log to files that
    # can then be read from the sidecar. To avoid a disk explosion, we need to
    # layer on some rotation. multilog is a tool that automatically writes its
    # stdin to rotated log files; the following line pipes stdout and stderr of
    # this process to separate multilog invocations. "n2" means to only store
    # one old log file -- the logs are being streamed out by Fluent Bit, so we
    # don't need to keep any more old ones around.
    exec > >(multilog n2 "$STDOUT_FILE-rotate")  2> >(multilog n2 "$STDERR_FILE-rotate")
fi

set -e

if [ "$#" -eq 1 ];
then
    exec /bin/sh -c $@
else
    exec $@
fi
