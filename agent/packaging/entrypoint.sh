#!/bin/bash

container_startup_script="/usr/local/bin/container_startup.sh"

if [ -f "$container_startup_script" ]; then
    chmod +x $container_startup_script
    $container_startup_script
    exit_code=$?
    if [ $exit_code -ne 0 ]; then
        echo "container_startup_script failed with exit code $exit_code" >&2
        exit 1
    fi
fi

exec /usr/bin/determined-agent "$@"
