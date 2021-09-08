#!/bin/bash

chmod +x /etc/determined/container_startup_script
/etc/determined/container_startup_script

/usr/bin/determined-agent "$@"
