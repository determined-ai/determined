#!/usr/bin/env bash

source /run/determined/task-logging-setup.sh

set -e

if [ "$#" -eq 1 ];
then
    exec /bin/sh -c "$@"
else
    exec "$@"
fi
