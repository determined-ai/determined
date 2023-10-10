.. _dynamic-agents-gcp:

#######################################
 Deploy Determined with Dynamic Agents
#######################################

This document describes how to install, configure, and upgrade a deployment of Determined with
dynamic agents on GCP. For an overview of the elastic infrastructure in Determined, visit
:ref:`elastic-infrastructure`.

*********************
 System Requirements
*********************

Compute Engine Project
======================

The Determined master and the Determined agents are intended to run in the same project.

Instance Labels
===============

When using dynamic agents on GCP, Determined identifies the Compute Engine instances that it is
managing using a configurable instance label (see :ref:`gcp-cluster-configuration` for details).
Administrators should be careful to ensure that this label is not used by other Compute Engine
instances that are launched outside of Determined; if that assumption is violated, unexpected
behavior may occur.

Compute Engine Images
=====================

-  The Determined master node will run on a custom image that will be shared with you by Determined
   AI.
-  Determined agent nodes will run on a custom image that will be shared with you by Determined AI.

Compute Engine Machine Types
============================

-  The Determined master node should be deployed on a Compute Engine instance with >= 2 CPUs (Intel
   Broadwell or later), 4GB of RAM, and 100GB of disk storage. This would be a Compute Engine
   ``n1-standard-2`` or more powerful.

.. _gcp-api-access:

GCP API Access
==============

-  The Determined master *needs* to run as a service account that has the permissions to manage
   Compute Engine instances. There are two options:

   #. Create a particular service account with the ``Compute Admin`` role. Then set the Determined
      master to use this account. See `Compute Engine IAM roles
      <https://cloud.google.com/compute/docs/access/iam>`__ for more details on how to configure the
      service account.

      -  In order for the Determined agent to be associated with a service account, the Determined
         master needs to have access to service accounts. Please ensure the service account of the
         Determined master has the ``Service Account User`` role.

      -  In order for the Determined agent to use a shared VPC, the service account that the master
         runs with needs to have the ``Compute Network User`` role.

   #. Use the default service account and add the ``Compute Engine: Read Write`` scope.

-  Optionally, the Determined agent may be associated with a service account.

.. note::

   Access scopes are the legacy method of specifying permissions for your instance. A best practice
   is to set the full cloud-platform access scope on the instance, then securely limit the service
   account's API access with Cloud IAM roles. See `Access Scopes
   <https://cloud.google.com/compute/docs/access/service-accounts#accesscopesiam>`__ for details.

.. _gcp-network-requirements:

Set up Internet Access
======================

-  The Determined Docker images are hosted on Docker Hub. Determined agents need access to Docker
   Hub for such tasks as building new images for user workloads.

-  If packages, data, or other resources needed by user workloads are hosted on the public Internet,
   Determined agents need to be able to access them. Note that agents can be :ref:`configured to use
   proxies <agent-network-proxy>` when accessing network resources.

-  For best performance, it is recommended that the Determined master and agents use the same
   physical network or VPC. When using VPCs on a public cloud provider, additional steps might need
   to be taken to ensure that instances in the VPC can access the Internet:

   -  On GCP, the instances need to have an external IP address, or a `GCP Cloud NAT
      <https://cloud.google.com/nat/docs/overview>`_ should be configured for the VPC.

   -  On AWS, the instances need to have a public IP address, and a `VPC Internet Gateway
      <https://docs.aws.amazon.com/vpc/latest/userguide/VPC_Internet_Gateway.html>`_ should be
      configured for the VPC.

Set up Firewall Rules
=====================

The firewall rules must satisfy the following network access requirements for the master and agents.

Master
------

-  Inbound TCP to the master's network port from the Determined agent instances, as well as all
   machines where developers want to use the Determined CLI or WebUI. The default port is ``8443``
   if TLS is enabled and ``8080`` if not.

-  Outbound TCP to all ports on the Determined agents.

Agents
------

-  Inbound TCP from all ports on the master to all ports on the agent.

-  Outbound TCP from all ports on the agent to the master's network port.

-  Outbound TCP to the services that host the Docker images, packages, data, and other resources
   that need to be accessed by user workloads.

   -  For example, if your data is stored on Amazon S3, ensure the firewall rules allow access to
      this data.

-  Inbound and outbound TCP on all ports to and from each Determined agent. The details are as
   follows:

   -  Inbound and outbound TCP ports 1734 and 1750 are used for synchronization between trial
      containers.

   -  Inbound and outbound TCP port 12350 is used for internal SSH-based communication between trial
      containers.

   -  Inbound and outbound TCP port 12355 is used for GLOO rendezvous between trial containers.

   -  Inbound and outbound ephemeral TCP ports in the range 1024-65536 are used for communication
      between trials via GLOO. This range is configured by the configuration parameter
      ``task_container_defaults.gloo_port_range`` inside ``master.yaml`` as described in the
      :ref:`cluster-configuration` guide.

   -  For every GPU on each agent machine, an inbound and outbound ephemeral TCP port in the range
      1024-65536 is used for communication between trials via NCCL. This range is configured by the
      configuration parameter ``task_container_defaults.nccl_port_range`` inside ``master.yaml`` as
      described in the :ref:`cluster-configuration` guide.

   -  Two additional ephemeral TCP ports in the range 1024-65536 are used for additional intra-trial
      communication between trial containers.

   -  For TensorBoards, an inbound and outbound TCP port between 2600-2900 is used to connect the
      master and the tensorboard container.

.. _gcp-gpu-requirements:

The following GPU types are supported by Determined:

-  ``nvidia-tesla-t4``
-  ``nvidia-tesla-p100``
-  ``nvidia-tesla-p4``
-  ``nvidia-tesla-v100``
-  ``nvidia-tesla-a100``

.. _gcp-cluster-configuration:

***********************
 Cluster Configuration
***********************

The Determined Cluster is configured with ``master.yaml`` file located at
``/usr/local/determined/etc`` on the Determined master instance. You need to configure GPU dynamic
agents in each resource pool. See :ref:`cluster-configuration` for details.

.. _gcp-attach-disk:

*************************************
 Attach a Disk To Each Dynamic Agent
*************************************

If your input data set is on a persistent disk, you can attach that disk to each dynamic agent by
using the base instance configuration and preparing commands. The following is an example
configuration. See `REST Resource: instances
<https://cloud.google.com/compute/docs/reference/rest/v1/instances/insert>`__ for the full list of
configuration options supported by GCP. See `Formatting and mounting a zonal persistent disk
<https://cloud.google.com/compute/docs/disks/add-persistent-disk#formatting>`__ for more examples of
formatting or mounting disks in GCP.

Here is an example master configuration to attach and mount a second disk to each dynamic agent.

.. code:: yaml

   provider:
     startup_script: |
                     lsblk
                     mkdir -p /mnt/disks/second
                     mount -o discard,defaults /dev/sdb1 /mnt/disks/second
                     lsblk
     type: gcp
     base_config:
       disks:
         - mode: READ_ONLY
           boot: false
           source: zones/<zone>/disks/<the name of the existing disk>
           autoDelete: false
     boot_disk_size: 200
     boot_disk_source_image: projects/<project>/global/images/<image name>

.. note::

   If a specific non-root user needs to access the disk, please run the tasks linked with the POSIX
   UID/GID of the user (See :ref:`run-as-user` for details.) and grant access to the corresponding
   UID/GID.

You can use the following command to validate if Determined tasks can read from the attached disk.

.. code::

   cat > command.yaml << EOF
   bind_mounts:
     - host_path: /mnt/disks/second
       container_path: /second
   EOF
   # Test attached read-only disk.
   det command run --config-file command.yaml ls -l /second

.. _gcp-pull-gcr:

*******************************
 Securely Pull Images from GCR
*******************************

If you have expensive operations to perform at startup, it can be useful to :ref:`add custom layers
<custom-env>` to the task images Determined provides. If you have store these images in a secure
registry, such as GCR, you can pull these images securely by using existing tooling like
`docker-credential-gcr <https://github.com/GoogleCloudPlatform/docker-credential-gcr>`__.

Here is an example master configuration of how to allow the agent to inherit the permissions of the
service account associated with a GCE instance, for accessing GCR.

.. code:: yaml

   provider:
     container_startup_script: |
         export HOME=/root
         apt-get update && apt-get install -y curl docker.io
         curl -fsSL "https://github.com/GoogleCloudPlatform/docker-credential-gcr/releases/download/v1.5.0/docker-credential-gcr_linux_amd64-1.5.0.tar.gz" \
               | tar xz --to-stdout > /usr/bin/docker-credential-gcr && chmod +x /usr/bin/docker-credential-gcr
         docker-credential-gcr configure-docker

.. note::

   This is an example of an operation that requires use of ``container_startup_script``. Because
   Docker credential helpers alter the Docker client configuration to depend on the helper binary by
   name, it must be installed and configured in the container.

**************
 Installation
**************

These instructions describe how to install Determined for the first time; for directions on how to
upgrade an existing Determined installation, see the :ref:`gcp-upgrades` section below.

Ensure that you are using the most up-to-date Determined images. Keep the image IDs handy as we will
need them later.

Master
======

To install the master, we will launch an instance from the Determined master image.

Let's start by navigating to the Compute Engine Dashboard of the GCP Console. Click "Create
Instance" and follow the instructions below:

#. Choose Machine Type: we recommend a ``n1-standard-2`` or more powerful.

#. Configure Boot Disk:

   #. Choose Boot Disk Image: find the Determined master image in "Images" and click "Select".

   #. Set Boot Disk Size: set ``Size`` to be at least 100GB. If you have a previous Determined
      installation that you are upgrading, you want to use the snapshot or existing disk. This disk
      will be used to store all your experiment metadata and checkpoints.

#. Configure Identity and API access: choose the ``service account`` according to
   :ref:`gcp-api-access`.

#. Configure Firewalls: choose or create a security group according to these
   :ref:`gcp-network-requirements`. Check off ``Allow HTTP traffic``.

#. Review and launch the instance.

#. SSH into the Determined master and edit the config at ``/usr/local/determined/etc/master.yaml``
   according to the guide on :ref:`cluster-configuration`.

#. Start the Determined master by entering ``make -C /usr/local/determined enable-master`` into the
   terminal.

Agent
=====

There is no installation needed for the agent. The Determined master will dynamically launch
Determined agent instances based on the :ref:`cluster-configuration`.

.. _gcp-upgrades:

**********
 Upgrades
**********

Upgrading an existing Determined installation with dynamic agents on GCP requires the same steps as
an installation without dynamic agents. See :ref:`upgrades`.
