.. _slurm-requirements:

###########################
 Installation Requirements
###########################

********************
 Basic Requirements
********************

Deploying the Determined HPC Launcher on Slurm/PBS has the following requirements.

-  The login node, admin node, and compute nodes must be installed and configured with one of the
   following Linux distributions:

   -  Red Hat® Enterprise Linux (RHEL) or CentOS 7.9
   -  RHEL or Rocky Linux® 8.5, 8.6
   -  RHEL 9
   -  SUSE® Linux Enterprise Server (SLES) 12 SP3 , 15 SP3, 15 SP4
   -  Ubuntu® 20.04, 22.04
   -  Cray OS (COS) 2.3, 2.4

   Note: More restrictive Linux distribution dependencies may be required by your choice of
   Slurm/PBS version and container runtime (Singularity/Apptainer®, Podman, or NVIDIA® Enroot).

-  Slurm 20.02 or greater (for versions greater than 22.05.2 see :ref:`slurm-known-issues`) or PBS
   2021.1.2 or greater.

-  Apptainer 1.0 or greater, Singularity 3.7 or greater, Enroot 3.4.0 or greater or PodMan 3.3.1 or
   greater.

-  A cluster-wide shared filesystem with consistent path names across the HPC cluster.

-  User and group configuration must be consistent across all nodes.

-  All nodes must be able to resolve the hostnames of all other nodes.

-  To run jobs with GPUs, the Nvidia or AMD drivers must be installed on each compute node.
   Determined requires a version greater than or equal to 450.80 of the Nvidia drivers. The Nvidia
   drivers can be installed as part of a CUDA installation but the rest of the CUDA toolkit is not
   required.

-  Determined supports the `active Python versions <https://endoflife.date/python>`__.

***********************
 Launcher Requirements
***********************

The launcher has the following additional requirements on the installation node:

-  Support for an RPM or Debian-based package installer
-  Java 1.8 or greater
-  Sudo is configured to process configuration files present in the ``/etc/sudoers.d`` directory
-  Access to the Slurm or PBS command line interface for the cluster
-  Access to a cluster-wide file system with a consistent path names across the cluster

.. _proxy-config-requirements:

**********************************
 Proxy Configuration Requirements
**********************************

If internet connectivity requires a use of a proxy, verify the following requirements:

-  Ensure that the proxy variables are defined in `/etc/environment` (or `/etc/sysconfig/proxy` on
   SLES).

-  Ensure that the `no_proxy` setting covers the login and admin nodes. If these nodes may be
   referenced by short names known only within the cluster, they must explicitly be included in the
   `no_proxy` setting.

-  If your experiment code communicates between compute nodes with a protocol that honors proxy
   environment variables, you should additionally include the names of all compute nodes in the
   `no_proxy` variable setting.

The HPC launcher imports `http_proxy`, `https_proxy`, `ftp_proxy`, `rsync_proxy`, `gopher_proxy`,
`socks_proxy`, `socks5_server`, and `no_proxy` from `/etc/environment` and `/etc/sysconfig/proxy`.
These environment variables are automatically exported in lowercase and uppercase into any launched
jobs and containers.

.. _slurm-config-requirements:

********************
 Slurm Requirements
********************

Determined should function with your existing Slurm configuration. The following steps are
recommended to optimize how Determined interacts with Slurm:

-  Enable Slurm for GPU Scheduling.

   Configure Slurm with `SelectType=select/cons_tres <https://slurm.schedmd.com/cons_res.html>`__.
   This enables Slurm to track GPU allocation instead of tracking only CPUs. When enabled,
   Determined submits batch jobs by specifying ``--gpus={slots_per_trial}``. If this is not
   available, you must change the :ref:`slurm section <cluster-configuration-slurm>`
   ``tres_supported`` option to ``false``.

-  Configure GPU Generic Resources (GRES).

   Determined works best when allocating GPUs. Information about what GPUs are available is
   available using GRES. You can use the `AutoDetect
   <https://slurm.schedmd.com/gres.html#AutoDetect>`__ feature to configure GPU GRES automatically.
   Otherwise, you should manually configure `GRES GPUs
   <https://slurm.schedmd.com/gres.html#GPU_Management>`__ such that Slurm can schedule nodes with
   the GPUs you want.

   For the automatic selection of nodes with GPUs, Slurm must be configured for ``GresTypes=gpu``
   and nodes with GPUs must have properly configured GRES indicating the presence of any GPUs. When
   enabled, Determined can ensure GPUs are available by specifying ``--gres=gpus:1``. If Slurm GRES
   cannot be properly configured, specify the :ref:`slurm section <cluster-configuration-slurm>`
   ``gres_supported`` option to ``false``, and it is the user's responsibility to ensure that GPUs
   will be available on nodes selected for the job using other configurations such as targeting a
   specific resource pool with only GPU nodes, or specifying a Slurm constraint in the experiment
   configuration.

-  Ensure homogeneous Slurm partitions.

   Determined maps Slurm partitions to Determined resource pools. It is recommended that the nodes
   within a partition be homogeneous for Determined to effectively schedule GPU jobs.

   -  A Slurm partition with GPUs is identified as a CUDA/ROCM resource pool. The type is inherited
      from the ``resource_manager.slot_type`` configuration. It can be also be specified-per
      partition using ``resource_manager.partition_overrides``

   -  A Slurm partition with no GPUs is identified as an AUX resource pool.

   -  The Determined default resource pool is set to the Slurm default partition. Override this
      default using the :ref:`slurm section <cluster-configuration-slurm>`
      ``default_compute_resource_pool`` or ``default_aux_resource_pool`` option.

-  Ensure the ``MaxNodes`` value for each partition is not less than the number of GPUs in the
   partition.

   Determined delegates node selection for a job to Slurm by specifying a node range
   (1-``slots_per_trial``). If ``slots_per_trial`` exceeds the ``MaxNodes`` value for the partition,
   the job will remain in state ``PENDING`` with reason code ``PartitionNodelimit``. Make sure that
   all partitions that have ``MaxNodes`` specified use a value larger than the number of GPUs in the
   partition.

-  Tune the Slurm configuration for Determined job preemption.

   Slurm preempts jobs using signals. When a Determined job receives SIGTERM, it begins a checkpoint
   and graceful shutdown. To prevent unnecessary loss of work, it is recommended to set ``GraceTime
   (secs)`` high enough to permit the job to complete an entire Determined ``scheduling_unit``.

   To enable GPU job preemption, use ``PreemptMode=REQUEUE`` or ``PreemptMode=REQUEUE``, because
   ``PreemptMode=SUSPEND`` does not release GPUs so does not allow a higher-priority job to access
   the allocated GPU resources.

.. _pbs-config-requirements:

******************
 PBS Requirements
******************

Determined should function with your existing PBS configuration. The following steps are recommended
to optimize how Determined interacts with PBS:

-  Configure PBS to manage GPU resources.

   Determined works best when allocating GPUs. By default, Determined selects compute nodes with
   GPUs using the option ``-select={slots_per_trial}:ngpus=1``. If PBS cannot be configured to
   identify GPUs in this manner, specify the :ref:`pbs section <cluster-configuration-slurm>`
   ``gres_supported`` option to ``false`` when configuring Determined, and it will then be the
   user's responsibility to ensure that GPUs will be available on nodes selected for the job using
   other configurations such as targeting a specific resource pool with only GPU nodes, or
   specifying a PBS constraint in the experiment configuration.

   PBS should be configured to provide the environment variable ``CUDA_VISIBLE_DEVICES``
   (``ROCR_VISIBLE_DEVICES`` for ROCm) using a PBS cgroup hook as described in the PBS
   Administrator's Guide. If PBS is not configured to set ``CUDA_VISIBLE_DEVICES``, Determined will
   utilize a single GPU on each node. To fully utilize multiple GPUs, you must either manually
   define ``CUDA_VISIBLE_DEVICES`` appropriately or provide the ``pbs.slots_per_node`` setting in
   your experiment configuration to indicate how many GPU slots are intended for Determined to use.

-  Ensure homogeneous PBS queues.

   Determined maps PBS queues to Determined resource pools. It is recommended that the nodes within
   a queue be homogeneous for Determined to effectively schedule GPU jobs.

   -  A PBS queue with GPUs is identified as a CUDA/ROCM resource pool. The type is inherited from
      the ``resource_manager.slot_type`` configuration. It can be also be specified per partition
      using ``resource_manager.partition_overrides``.

   -  A PBS queue with no GPUs is identified as an AUX resource pool.

   -  The Determined default resource pool is set to the PBS default queue. Override this default
      using the :ref:`pbs section <cluster-configuration-slurm>` ``default_compute_resource_pool``
      or ``default_aux_resource_pool`` option.

-  Tune the PBS configuration for Determined job preemption.

   PBS supports a wide variety of criteria to trigger job preemption, and you may use any per your
   system and job requirements. Once a job is identified for preemption, PBS supports four different
   options for job preemption which are specified via the ``preemption_order`` scheduling parameter.
   The preemption order value is ``'SCR'``. The preemption methods are specified by the following
   letters:

   ``S`` - Suspend the job.
      This is not applicable for GPU jobs.

   ``C`` - Checkpoint the job.
      This requires a custom checkpoint script is added to PBS.

   ``R`` - Requeue the job.
      Determined does not support the re-queueing of a task. Determined jobs specify the ``-r n``
      option to PBS to prevent this case.

   ``D`` - Delete the job.
      Determined jobs support this option without configuration.

   Given those options, the simplest path to enable Determined job preemption is by including ``D``
   in the ``preemption_order``. You may include ``R`` in the ``preemption_order``, but it is
   disabled for Determined jobs. You may include ``C`` to the ``preemption_order`` if you
   additionally configure a checkpoint script. Refer to the PBS documentation for details. If you
   choose to implement a checkpoint script, you may initiate a Determined checkpoint by sending a
   ``SIGTERM`` signal to the Determined job. When a Determined job receives a ``SIGTERM``, it begins
   a checkpoint and graceful shutdown. To prevent unnecessary loss of work, it is recommended that
   you wait for at least one Determined ``scheduling_unit`` for the job to complete after sending
   the ``SIGTERM``. If after that period of time the job has not terminated, then send a ``SIGKILL``
   to forcibly release all resources.

.. _singularity-config-requirements:

************************************
 Singularity/Apptainer Requirements
************************************

Singularity/Apptainer is the recommended container runtime for Determined on HPC clusters. Apptainer
is a fork of Singularity 3.8 and provides both the ``apptainer`` and ``singularity`` commands. For
purposes of this documentation, you can consider all references to Singularity to also apply to
Apptainer. The Determined launcher interacts with Singularity/Apptainer using the ``singularity``
command.

Singularity has numerous options that may be customized in the ``singularity.conf`` file. Determined
has been verified using the default values and therefore does not require any special configuration
on the compute nodes of the cluster.

.. _podman-config-requirements:

*********************
 PodMan Requirements
*********************

When Determined is configured to use PodMan, the containers are launched in `rootless mode
<https://docs.podman.io/en/latest/markdown/podman.1.html#rootless-mode>`__. Your HPC cluster
administrator should have completed most of the configuration for you, but there may be additional
per-user configuration that is required. Before attempting to launch Determined jobs, verify that
you can run simple PodMan containers on a compute node. For example:

.. code:: bash

   podman run hello-world

If you are unable to do that successfully, then one or more of the following configuration changes
may be required in your ``$HOME/.config/containers/storage.conf`` file:

#. PodMan does not support rootless container storage on distributed file systems (e.g. NFS, Lustre,
   GPSF). On a typical HPC cluster, user directories are on a distributed file system and the
   default container storage location of ``$HOME/.local/share/containers/storage`` is therefore not
   supported. If this is the case on your HPC cluster, configure the ``graphroot`` option in your
   ``storage.conf`` to specify a local file system available on compute nodes. Alternatively, you
   can request that your system administrator configure the ``rootless_storage_path`` in
   ``/etc/containers/storage.conf`` on all compute nodes.

#. PodMan utilizes the directory specified by the environment variable ``XDG_RUNTIME_DIR``.
   Normally, this is provided by the login process. Slurm and PBS, however, do not provide this
   variable when launching jobs on compute nodes. When ``XDG_RUNTIME_DIR`` is not defined, PodMan
   attempts to create the directory ``/run/user/$UID`` for this purpose. If ``/run/user`` is not
   writable by a non-root user, then PodMan commands will fail with a permission error. To avoid
   this problem, configure the ``runroot`` option in your ``storage.conf`` to a writeable local
   directory available on all compute nodes. Alternatively, you can request your system
   administrator to configure the ``/run/user`` to be user-writable on all compute nodes.

Create or update ``$HOME/.config/containers/storage.conf`` as required to resolve the issues above.
The example ``storage.conf`` file below uses the file system ``/tmp``, but there may be a more
appropriate file system on your HPC cluster that you should specify for this purpose.

.. code:: docker

   [storage]
   driver = "overlay"
   graphroot = "/tmp/$USER/storage"
   runroot = "/tmp/$USER/run"

Any changes to your ``storage.conf`` should be applied using the command:

.. code:: bash

   podman system migrate

.. _enroot-config-requirements:

*********************
 Enroot Requirements
*********************

Install and configure Enroot on all compute nodes of your cluster as per the `Enroot Installation
instructions <https://github.com/NVIDIA/enroot/blob/master/doc/installation.md>`__ for your
platform. There may be additional per-user configuration that is required.

#. Enroot utilizes the directory ``${ENROOT_RUNTIME_PATH}`` (with default value
   ``${XDG_RUNTIME_DIR}/enroot``) for temporary files. Normally ``XDG_RUNTIME_DIR`` is provided by
   the login process, but Slurm and PBS do not provide this variable when launching jobs on compute
   nodes. When neither ENROOT_RUNTIME_PATH/XDG_RUNTIME_DIR is defined, Enroot attempts to create the
   directory /run/enroot for this purpose. This typically fails with a permission error for any
   non-root user. Select one of the following alternatives to ensure that ``XDG_RUNTIME_DIR`` or
   ``ENROOT_RUNTIME_PATH`` is defined and points to a user-writable directory when Slurm/PBS jobs
   are launched on the cluster.

   -  Have your HPC cluster administrator configure Slurm/PBS to provide ``XDG_RUNTIME_DIR``, or
         change the default ``ENROOT_RUNTIME_PATH`` defined in ``/etc/enroot/enroot.conf`` on each
         node in your HPC cluster.

   -  If using Slurm, provide an ``ENROOT_RUNTIME_PATH`` definition in
      ``task_container_defaults.environment_variables`` in master.yaml.

      .. code:: yaml

         task_container_defaults:
            environment_variables:
               - ENROOT_RUNTIME_PATH=/tmp/$(whoami)

   -  If using Slurm, provide an ``ENROOT_RUNTIME_PATH`` definition in your experiment
      configuration.

#. Unlike Singularity or PodMan, you must manually download the docker image file to the local file
   system (``enroot import``) and then each user must create an Enroot container using that image
   (``enroot create``). When the HPC launcher generates the enroot command for a job, it
   automatically applies the same transformation to the name that Enroot does on import (``/`` and
   ``:`` characters are replaced with ``+``) to enable docker mage references to match the
   associated Enroot container. The following shell commands will download and then create an Enroot
   container for the current user. If other users have read access to ``/shared/enroot/images``,
   they need only perform the ``enroot create`` step to make the container available for their use.

   .. code:: bash

      image=determinedai/environments:cuda-11.3-pytorch-1.10-tf-2.8-gpu-24586f0
      cd /shared/enroot/images
      enroot import docker://$image
      enroot create /shared/enroot/images/${image//[\/:]/\+}.sqsh

#. The Enroot container storage directory for the user ``${ENROOT_CACHE_PATH}`` (which defaults to
   ``$HOME/.local/share/enroot``) must be accessible on all compute nodes.
