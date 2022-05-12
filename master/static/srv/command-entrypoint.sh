#!/usr/bin/env bash

source /run/determined/task-signal-handling.sh
source /run/determined/task-logging-setup.sh
trap 'source /run/determined/task-logging-teardown.sh' EXIT

set -e

trap_and_forward_signals
if [ "$#" -eq 1 ];
then
    /bin/sh -c "$@" &
else
    "$@" &
fi
wait_and_handle_signals $!
