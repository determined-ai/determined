#!/usr/bin/env bash

if [[ ${workload_manager} == pbs ]]; then
    sudo sed -i "s|^PBS_SERVER=.*|PBS_SERVER=$(hostname -f)|" /etc/pbs.conf
    sudo sed -i "s|^PBS_START_MOM=.*|PBS_START_MOM=1|" /etc/pbs.conf

    sudo systemctl start pbs
fi
