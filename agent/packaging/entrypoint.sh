#!/bin/bash

chmod +x /usr/local/determined/container_startup_script
/usr/local/determined/container_startup_script
exit_code=$?
if [ $exit_code -ne 0 ]; then
    echo "container_startup_script failed with exit code $exit_code" >&2
    exit 1
fi

exec /usr/bin/determined-agent "$@"
