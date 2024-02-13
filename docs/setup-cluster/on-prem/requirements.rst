.. _requirements:

##############
 Requirements
##############

.. _system-requirements:

*********************
 System Requirements
*********************

A Determined cluster has the following requirements.

Software
========

-  The Determined agent and master nodes must be configured with Ubuntu 20.04 or later, Enterprise
   Linux 7 (such as AlmaLinux, Red Hat Enterprise Linux, or Rocky Linux), or macOS 10.13 or later.

-  The agent nodes must have :ref:`Docker installed <install-docker>`.

-  To run jobs with GPUs, the NVIDIA drivers must be installed on each Determined agent. Determined
   requires a version greater than or equal to 450.80 of the NVIDIA drivers. The NVIDIA drivers can
   be installed as part of a CUDA installation but the rest of the CUDA toolkit is not required.

-  Determined supports the `active Python versions <https://endoflife.date/python>`__.

Hardware
========

-  The Determined master node should be configured with at least four Intel Broadwell or later CPU
   cores, 8GB of RAM, and 200GB of free disk space. The Determined master node does not need GPUs.

-  Each Determined agent node should be configured with at least two Intel Broadwell or later CPU
   cores, 4GB of RAM, and 50GB of free disk space. If you are using GPUs, NVIDIA GPUs with compute
   capability 6.0 or greater are required. These include P100, V100, A100, RTX 2080 Ti, RTX 3090,
   TITAN X, and TITAN XP.

Most of the disk space required by the master is because of the experiment metadata database. If
PostgreSQL is set up on a different machine, the disk space requirements for the master are minimal
(~100MB).

.. _install-docker:

****************
 Install Docker
****************

Docker is a dependency of several Determined system components. For example, every agent node must
have Docker installed to run containerized workloads.

.. _install-docker-on-linux:

Install on Linux
================

#. Install Docker. Docker version 20.10 or later is required on the machine where the agent is
   running.

   On Ubuntu:

   .. code:: bash

      sudo apt update && sudo apt install -y ca-certificates curl gnupg
      sudo install -m 0755 -d /etc/apt/keyrings
      curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg
      sudo chmod a+r /etc/apt/keyrings/docker.gpg
      echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu $(. /etc/os-release && echo "$VERSION_CODENAME") stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null

      sudo apt update && sudo apt install -y --no-install-recommends docker-ce
      sudo usermod -aG docker $USER
      sudo systemctl reload docker

   On Enterprise Linux:

   .. code:: bash

      sudo dnf install -y device-mapper-persistent-data lvm2
      sudo dnf config-manager --add-repo https://download.docker.com/linux/centos/docker-ce.repo

      sudo dnf install -y docker-ce
      sudo usermod -aG docker $USER
      sudo systemctl start docker

#. If the machine has GPUs that you want to use with Determined, install the NVIDIA Container
   Toolkit to allow Docker to run containers that use the GPUs. For more information, see the
   `NVIDIA documentation <https://github.com/NVIDIA/nvidia-docker>`__.

   On Ubuntu:

   .. code:: bash

      curl -fsSL https://nvidia.github.io/libnvidia-container/gpgkey | sudo gpg --dearmor -o /usr/share/keyrings/nvidia-container-toolkit-keyring.gpg
      curl -s -L https://nvidia.github.io/libnvidia-container/stable/deb/nvidia-container-toolkit.list | sed 's#deb https://#deb [signed-by=/usr/share/keyrings/nvidia-container-toolkit-keyring.gpg] https://#g' | sudo tee /etc/apt/sources.list.d/nvidia-container-toolkit.list > /dev/null

      sudo apt update
      sudo apt install -y nvidia-container-toolkit
      sudo systemctl restart docker

   On Enterprise Linux:

   .. code:: bash

      curl -s -L https://nvidia.github.io/libnvidia-container/stable/rpm/nvidia-container-toolkit.repo | sudo tee /etc/yum.repos.d/nvidia-container-toolkit.repo

      sudo dnf install -y nvidia-container-toolkit
      sudo systemctl restart docker

#. Log out and start a new terminal session.

#. Verify that the current user is in the ``docker`` group and, if the machine has GPUs, that Docker
   can start a container using them:

   .. code:: bash

      groups
      docker run --gpus all --rm debian:10-slim nvidia-smi

#. If you are using Enterprise Linux, `enable the journalctl log messages persistent storage
   <https://unix.stackexchange.com/a/159390>`_ so logs are saved on machine reboot:

   .. code:: bash

      sudo mkdir /var/log/journal
      sudo systemd-tmpfiles --create --prefix /var/log/journal
      sudo systemctl restart systemd-journald

.. _install-docker-on-macos:

Install on macOS
================

#. Install Docker for macOS by following the `Docker documentation
   <https://docs.docker.com/desktop/mac/install/>`__. The Docker documentation describes system
   requirements, chipset dependencies, and installation steps.

#. Start Docker:

   .. code:: bash

      open /Applications/Docker.app

Docker on macOS does not support containers that use GPUs. Because of this, macOS Determined agents
are only able to run CPU-based workloads.

.. _install-docker-on-wsl:

Install on Windows Subsystem for Linux (WSL)
============================================

Follow the steps for :ref:`installing Docker on Linux <install-docker-on-linux>`.
