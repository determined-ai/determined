#!/usr/bin/env bash

exec >/var/log/startup-script.log 2>&1

# Startup and configuration must be done upon startup of VM since `hostname -f`
# is different inside the packer instance (upon creation of image).
if [[ ${WORKLOAD_MANAGER} == pbs ]]; then
    sudo sed -i "s|^PBS_SERVER=.*|PBS_SERVER=$(hostname -f)|" /etc/pbs.conf
    sudo sed -i "s|^PBS_START_MOM=.*|PBS_START_MOM=1|" /etc/pbs.conf

    sudo systemctl start pbs

    # Wait for the PBS service to start
    while ! sudo systemctl is-active --quiet pbs; do
        echo "Waiting for PBS service to start..."
        sleep 2
    done
    echo "PBS service started"

    # Enable job history
    echo "set server job_history_enable = true" | sudo /opt/pbs/bin/qmgr
    echo "Set job_history_enable to true"
fi
