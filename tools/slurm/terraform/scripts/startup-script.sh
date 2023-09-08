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
else
    # Update the Gres expression for the local node in slurm.conf
    echo "Updating the Gres configuration in /etc/slurm/slurm.conf from nvidia-smi information..."
    nvidia-smi
    STATUS=$?
    GPU_COUNT=$(nvidia-smi --query-gpu=index --format=csv,noheader | wc -l)
    GRES_TYPE=$(nvidia-smi -i 0 --query-gpu=name --format=csv,noheader | cut -f 1 -d ' ' | tr '[:upper:]' '[:lower:]')
    # If nvidia-smi succeeded, and we have any GPUs, then configure the Gres for them.
    if [[ $${STATUS} == 0 && $${GPU_COUNT} -gt 0 ]]; then
        # This is an ansible template file so env references using braces need to be quoted with an
        # additional $ to delay evaluation until script execution time.
        GRES="Gres=gpu:$${GRES_TYPE}:$${GPU_COUNT}"
        echo "Detected $${GRES}"
        sudo sed -i "s/Gres=.*//" /etc/slurm/slurm.conf
        sudo sed -i "s/^NodeName=.*/& $${GRES}/" /etc/slurm/slurm.conf
        echo "grep NodeName /etc/slurm/slurm.conf"
        grep NodeName /etc/slurm/slurm.conf
        # Slurm is already started, need to reconfigure so
        sudo scontrol reconfigure
        sinfo
    fi
fi
