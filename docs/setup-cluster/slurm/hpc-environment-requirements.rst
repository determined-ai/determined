.. _hpc-environment-requirements:

##############################
 HPC Environment Requirements
##############################

This document describes how to prepare your environment for installing Determined on an HPC cluster
managed by Slurm or PBS workload managers.

.. include:: ../../_shared/tip-keep-install-instructions.txt

**************************
 Environment Requirements
**************************

Hardware Requirements
=====================

The recommended requirements for the admin node are:

-  1 admin node for the master, the database, and the launcher with the following specs:
      -  16 cores
      -  32 GB of memory
      -  1 TB of disk space (depends on the database, see "Database Requirements" section below)

The minimal requirements are:

-  1 admin node with 8 cores, 16 GB of memory, and 200 GB of disk space

.. note::

   While the node can be virtual, a physical one is preferred.

Network Requirements
====================

The admin node requires the following network configurations:

Admin Node
----------

**Ports:** 8080, 8443 **Type:** TCP **Description:** Provide HTTP(S) access to the master node for
web UI access and agent API access

.. note::

   Ensure these ports are open in your firewall settings to allow proper communication with the
   admin node.

Additional Requirements:

   -  The admin node must reach the HPC shared area (the scratch file system).
   -  Recommended: 10 Gbps Ethernet link between the admin node and the HPC worker nodes.
   -  Minimal: 1 Gbps Ethernet link.

.. important::

   The admin node must be connected to the Internet to download container images and Python
   packages. If Internet access is not possible, the local container registry and package repository
   must be filled manually with external data.

Storage Requirements
====================

Determined requires shared storage for experiment checkpoints, container images, datasets, and
pre-trained models. All worker nodes connected to the cluster must be able to access it. The storage
can be a network file system (like VAST, Ceph FS, Gluster FS, Lustre) or a bucket (on cloud or
on-prem if it exposes an S3 API).

Space requirements depend on the model complexity/size:

-  10-30 TB of HDD space for small models (up to 1GB in size)
-  20-60 TB of SSD space for medium to large models (more than 1GB in size)

Software Requirements
=====================

The following software components are required:

+------------------------+----------------------------------+------------------+
| Component              | Version                          | Installation     |
|                        |                                  | Node             |
+========================+==================================+==================+
| Operating System       | RHEL 8.5+ or 9.0+ SLES 15 SP3+   | Admin            |
|                        | Ubuntu 22.04+                    |                  |
+------------------------+----------------------------------+------------------+
| Java                   | >= 1.8                           | Admin            |
+------------------------+----------------------------------+------------------+
| Python                 | >= 3.8                           | Admin            |
+------------------------+----------------------------------+------------------+
| Podman                 | >= 4.0.0                         | Admin            |
+------------------------+----------------------------------+------------------+
| PostgreSQL             | 10 (RHEL 8), 13 (RHEL 9), 14     | Admin            |
|                        | (Ubuntu 22.04) or later          |                  |
+------------------------+----------------------------------+------------------+
| HPC client packages    | Same as login nodes              | Admin            |
+------------------------+----------------------------------+------------------+
| Container runtime      | Singularity >= 3.7 (or Apptainer | Workers          |
|                        | >= 1.0) Podman >= 3.3.1 Enroot   |                  |
|                        | >= 3.4.0                         |                  |
+------------------------+----------------------------------+------------------+
| HPC scheduler          | Slurm >= 20.02 (excluding        | Workers          |
|                        | 22.05.5 - 22.05.8) PBS >=        |                  |
|                        | 2021.1.2                         |                  |
+------------------------+----------------------------------+------------------+
| NVIDIA drivers         | >= 450.80                        | Workers          |
+------------------------+----------------------------------+------------------+

Database Requirements
=====================

The solution requires PostgreSQL 13 or later, which will be installed on the admin node. We
recommend using the latest available version of PostgreSQL for optimal support and security. The
required disk space for the database is estimated as follows:

-  200 GB on small systems (less than 15 workers) or big systems if the experiment logs are sent to
   Elasticsearch
-  16 GB/worker on big systems that store experiment logs inside the database

****************************
 Installation Prerequisites
****************************

Before proceeding with the installation, ensure that:

-  The operating system is installed along with the HPC client packages (a clone of an existing
   login node could be made if the OS is the same or similar)
-  The node has Internet connectivity
-  The node has the shared file system mounted on /scratch
-  Java is installed
-  Podman is installed

A dedicated OS user named ``determined`` should be created on the admin node. This user should:

-  Belong to the ``determined`` group
-  Be able to run HPC jobs
-  Have sudo permissions for specific commands (see :ref:`hpc-security-considerations` for details)

.. note::

   All subsequent installation steps assume the use of the ``determined`` user or root access.

For detailed installation steps, including OS-specific instructions and configuration, refer to the
:ref:`install-on-slurm` document.

Internal Task Gateway
=====================

As of version 0.34.0, Determined supports the Internal Task Gateway feature for Kubernetes. This
feature enables Determined tasks running on remote Kubernetes clusters to be exposed to the
Determined master and proxies. If you're using a hybrid setup with both Slurm/PBS and Kubernetes,
this feature might be relevant for your configuration.

.. important::

   Enabling this feature exposes Determined tasks to the outside world. Implement appropriate
   security measures to restrict access to exposed tasks and secure communication between the
   external cluster and the main cluster.
