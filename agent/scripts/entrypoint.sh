#!/bin/bash

chmod +x /usr/local/determined/container_startup_script
/usr/local/determined/container_startup_script

/usr/bin/determined-agent "$@"
