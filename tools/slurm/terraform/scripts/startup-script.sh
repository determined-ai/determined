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

    # enable ngpus resource https://rndwiki-pro.its.hpecorp.net/display/AT/casablanca-login2%3A+configuring+GPUs+with+PBS
    sudo sed -i 's/^resources: "ncpus/resources: "ngpus, ncpus/' /var/spool/pbs/sched_priv/sched_config
    sudo /opt/pbs/bin/qmgr -c "create resource ngpus type=long,flag=nh"
    sudo /opt/pbs/bin/qmgr -c "export hook pbs_cgroups application/x-config default" >/tmp/old_pbs_cgroups.json
    sudo cat /tmp/old_pbs_cgroups.json \
        | jq '. += {"nvidia-smi":"/usr/bin/nvidia-smi"}' \
        | jq '.cgroup.devices += {"enabled":true}' \
        | jq '.cgroup.memory += {"enforce_default":false}' \
        | jq '.cgroup.devices.allow += ["c *:* rwm", ["nvidiactl","rwm","*"]]' \
        | jq '.cgroup.cpuset += {"enabled":true}' >/tmp/pbs_cgroups.json
    sudo /opt/pbs/bin/qmgr -c "import hook pbs_cgroups application/x-config default /tmp/pbs_cgroups.json"
    echo INFO: Updated pbs_cgroups.json - pbs_cgroups hook enbled if GPUs available below
    sudo cat /tmp/pbs_cgroups.json

    # Enable job history
    echo "set server job_history_enable = true" | sudo /opt/pbs/bin/qmgr
    echo "Set job_history_enable to true"

    # Enable cgroups & configure PBS accel_type if GPUs available
    nvidia-smi
    STATUS=$?
    GPU_COUNT=$(nvidia-smi --query-gpu=index --format=csv,noheader | wc -l)
    if [[ $${STATUS} == 0 && $${GPU_COUNT} -gt 0 ]]; then
        sudo /opt/pbs/bin/qmgr -c "create resource accel_type type=string_array,flag=h"
        GRES_TYPE=$(nvidia-smi -i 0 --query-gpu=name --format=csv,noheader | cut -f 1 -d ' ' | tr '[:upper:]' '[:lower:]')
        sudo /opt/pbs/bin/qmgr -c "set node $(hostname) resources_available.accel_type=$${GRES_TYPE}"
        echo "Configured accel_type $${GRES_TYPE}"
        sudo /opt/pbs/bin/qmgr -c "set hook pbs_cgroups enabled=True"
    else
        echo "No GPUs available.  Leaving pbs_cgroups disabled."
    fi

    # Restart to apply GPU changes
    sudo systemctl restart pbs
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
