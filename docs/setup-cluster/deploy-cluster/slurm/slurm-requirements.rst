.. _slurm-requirements:

###########################
 Installation Requirements
###########################

********************
 Basic Requirements
********************

To deploy the Determined HPC Launcher on Slurm/PBS, the following requirements must be met.

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

-  Slurm 20.02 or greater (excluding 22.05.5 through at least 22.05.8 - see
   :ref:`slurm-known-issues`) or PBS 2021.1.2 or greater.

-  Apptainer 1.0 or greater, Singularity 3.7 or greater, Enroot 3.4.0 or greater or Podman 3.3.1 or
   greater.

-  A cluster-wide shared filesystem with consistent path names across the HPC cluster.

-  User and group configuration must be consistent across all nodes.

-  All nodes must be able to resolve the hostnames of all other nodes.

-  To run jobs with GPUs, the NVIDIA or AMD drivers must be installed on each compute node.
   Determined requires a version greater than or equal to 450.80 of the NVIDIA drivers. The NVIDIA
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
-  Access to the Slurm or PBS command-line interface for the cluster
-  Access to a cluster-wide file system with a consistent path names across the cluster

.. _proxy-config-requirements:

**********************************
 Proxy Configuration Requirements
**********************************

If internet connectivity requires a use of a proxy, verify the following requirements:

-  Ensure that the proxy variables are defined in ``/etc/environment`` (or ``/etc/sysconfig/proxy``
   on SLES).

-  Ensure that the `no_proxy` setting covers the login and admin nodes. If these nodes may be
   referenced by short names known only within the cluster, they must explicitly be included in the
   `no_proxy` setting.

-  If your experiment code communicates between compute nodes with a protocol that honors proxy
   environment variables, you should additionally include the names of all compute nodes in the
   `no_proxy` variable setting.

The HPC launcher imports `http_proxy`, `https_proxy`, `ftp_proxy`, `rsync_proxy`, `gopher_proxy`,
`socks_proxy`, `socks5_server`, and `no_proxy` from ``/etc/environment`` and
``/etc/sysconfig/proxy``. These environment variables are automatically exported in lowercase and
uppercase into any launched jobs and containers.

.. _slurm-config-requirements:

********************
 Slurm Requirements
********************

Determined should function with your existing Slurm configuration. To optimize how Determined
interacts with Slurm, we recommend the following steps:

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

   -  A Slurm partition with GPUs is identified as a CUDA/ROCm resource pool. The type is inherited
      from the ``resource_manager.slot_type`` configuration. It can be also be specified-per
      partition using ``resource_manager.partition_overrides``

   -  A Slurm partition with no GPUs is identified as an AUX resource pool.

   -  The Determined default resource pool is set to the Slurm default partition. Override this
      default using the :ref:`slurm section <cluster-configuration-slurm>`
      ``default_compute_resource_pool`` or ``default_aux_resource_pool`` option.

   -  If a Slurm partition is not homogeneous, you may create a resource pool that provides
      homogenous resources out of that partition using a custom resource pool. Configure a
      :ref:`resource pool <cluster-resource-pools>` with ``provider_type: hpc``, specify the
      underlying Slurm partition name to receive the job and include a :ref:`task_container_defaults
      <master-task-container-defaults>` section with the necessary ``slurm`` options to select the
      desired homogenous set of resources from that partition.

-  Ensure the ``MaxNodes`` value for each partition is not less than the number of GPUs in the
   partition.

   Determined delegates node selection for a job to Slurm by specifying a node range
   (1-``slots_per_trial``). If ``slots_per_trial`` exceeds the ``MaxNodes`` value for the partition,
   the job will remain in state ``PENDING`` with reason code ``PartitionNodelimit``. Make sure that
   all partitions that have ``MaxNodes`` specified use a value larger than the number of GPUs in the
   partition.

-  Enable multiple jobs per compute node.

   Determined uses GPU or CPU resource requests to Slurm. When Slurm schedules jobs, however, it
   also considers the memory requirements of the job. In order to enable multiple jobs to be
   scheduled on a node concurrently, configuration is required in `slurm.conf
   <https://slurm.schedmd.com/slurm.conf.html>`__.

   The default memory allocated for a job is ``UNLIMITED``. This prevents multiple jobs from
   executing on the same node unless this value is reduced. The default memory allocation for a job
   is derived from one of the `slurm.conf` configuration variables ``DefMemPerNode``,
   ``DefMemPerGPU``, or ``DefMemPerCPU``. In order to enable individual GPUs/CPUs scheduling by
   default configure ``DefMemPerNode`` (which provides a total amount of memory for each job) or
   ``DefMemPerGPU`` and ``DefMemPerCPU`` (which derives the memory allocation from the number of GPU
   or CPU associated with the job). Configure one or more of these values to reduce the default
   memory allocation and enable jobs to divide up the available memory on compute nodes.

   An alternative to changing the default memory configuration via `slurm.conf
   <https://slurm.schedmd.com/slurm.conf.html>`__, is to provide explicit options on each job via
   the Determined configuration (:ref:`task_container_defaults <master-task-container-defaults>`,
   :ref:`resource pool <cluster-resource-pools>` configuration, or experiment configuration
   :ref:`slurm.sbatch_args <sbatch-args>`).

   For details about how those requests are derived, see :ref:`hpc_launching_architecture`.

-  Enable resource separation using cgroups.

   While Slurm always allocates distinct resources for each job, by default there is no enforced
   separation when the resources are co-located in the same compute node. Such enforcement can be
   enabled using `cgroups <https://slurm.schedmd.com/cgroups.html>`__. GPU allocation is
   communicated to the application via the environment variables ``CUDA_VISIBLE_DEVICES`` or
   ``ROCR_VISIBLE_DEVICES``. Determined uses those specifications to utilize only the GPU resources
   scheduled by Slurm for the job, but CPU and memory have no enforcement. If desired, you can
   enable such enforcement with the Slurm `cgroups <https://slurm.schedmd.com/cgroups.html>`__
   configuration. Enable cgroups support in `slurm.conf
   <https://slurm.schedmd.com/slurm.conf.html>`__, then enable enforcement of specific resource
   classes in `cgroup.conf <https://slurm.schedmd.com/cgroup.conf.html>`__ (``ConstrainCores`` for
   CPU, ``ConstrainDevices`` for GPU, and ``ConstrainRAMSpace`` for memory).

-  Tune the Slurm configuration for Determined job preemption.

   Slurm preempts jobs using signals. When a Determined job receives SIGTERM, it begins a checkpoint
   and graceful shutdown. To prevent unnecessary loss of work, it is recommended to set ``GraceTime
   (secs)`` high enough to permit the job to complete an entire Determined ``scheduling_unit``.

   To enable GPU job preemption, use ``PreemptMode=CANCEL`` or ``PreemptMode=REQUEUE``, because
   ``PreemptMode=SUSPEND`` does not release GPUs so does not allow a higher-priority job to access
   the allocated GPU resources. Determined manages the requeue of a successfully preempted job so
   even with ``PreemptMode=REQUEUE``, the Slurm job will be canceled and resubmitted.

.. _pbs-config-requirements:

******************
 PBS Requirements
******************

Determined should function with your existing PBS configuration. To optimize how Determined
interacts with PBS, we recommend the following steps:

-  Enable PBS to store job history.

   To ensure successful job completion detection by the HPC launcher, it is crucial to have the job
   history enabled in PBS. In the absence of proper configuration, the HPC launcher would fail to
   resolve the status or information of a job post-completion.

   PBS administrators can employ the following commands to set and confirm the value of
   ``job_history_enable``:

   -  Set the value of ``job_history_enable``.

      .. code:: bash

         sudo qmgr -c "set server job_history_enable = True"

   -  Verify that the new ``job_history_enable`` value is now set.

      .. code:: bash

         qmgr -c "print server job_history_enable"

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

-  Configure PBS to report GPU Accelerator type.

   It is recommended that PBS administrators set the value for ``resources_available.accel_type`` on
   each node that contains an accelerator. Otherwise, the Cluster tab on the Determined Web UI will
   show ``unconfigured`` for the ``Accelerator`` field in the Resource Pool information.

   PBS administrators can use the following set of commands to set the value of
   ``resources_available.accel_type`` on a single node:

   -  Check if the ``resources_available.accel_type`` value is set.

      .. code:: bash

         pbsnodes -v node001 | grep resources_available.accel_type

   -  If required, set the desired value for ``resources_available.accel_type``.

      .. code:: bash

         sudo qmgr -c "set node node001 resources_available.accel_type=tesla"

   -  When there are multiple types of GPUs on the node, use a comma-separated value.

      .. code:: bash

         sudo qmgr -c "set node node001 resources_available.accel_type=tesla,kepler"

   -  Verify that the ``resources_available.accel_type`` value is now set.

      .. code:: bash

         pbsnodes -v node001 | grep resources_available.accel_type

   Repeat the above steps to set the ``resources_available.accel_type`` value for every node
   containing GPU. Once the ``resources_available.accel_type`` value is set for all the necessary
   nodes, admins can verify the Accelerator field on the Cluster tab of the Web UI.

-  Ensure homogeneous PBS queues.

   Determined maps PBS queues to Determined resource pools. It is recommended that the nodes within
   a queue be homogeneous for Determined to effectively schedule GPU jobs.

   -  A PBS queue with GPUs is identified as a CUDA/ROCm resource pool. The type is inherited from
      the ``resource_manager.slot_type`` configuration. It can be also be specified per partition
      using ``resource_manager.partition_overrides``.

   -  A PBS queue with no GPUs is identified as an AUX resource pool.

   -  The Determined default resource pool is set to the PBS default queue. Override this default
      using the :ref:`pbs section <cluster-configuration-slurm>` ``default_compute_resource_pool``
      or ``default_aux_resource_pool`` option.

   -  If a PBS queue is not homogeneous, you may create a resource pool that provides homogenous
      resources out of that queue using a custom resource pool. Configure a :ref:`resource pool
      <cluster-resource-pools>` with ``provider_type: hpc``, specify the underlying PBS queue name
      to receive the job and include a :ref:`task_container_defaults
      <master-task-container-defaults>` section with the necessary ``pbs`` options to select the
      desired homogenous set of resources from that queue.

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
 Apptainer/Singularity Requirements
************************************

Apptainer/Singularity is the recommended container runtime for Determined on HPC clusters. Apptainer
is a fork of Singularity 3.8 and provides both the ``apptainer`` and ``singularity`` commands. For
purposes of this documentation, you can consider all references to Singularity to also apply to
Apptainer. The Determined launcher interacts with Apptainer/Singularity using the ``singularity``
command.

Singularity has numerous options that may be customized in the ``singularity.conf`` file. Determined
has been verified using the default values and therefore does not require any special configuration
on the compute nodes of the cluster.

.. _podman-config-requirements:

*********************
 Podman Requirements
*********************

When Determined is configured to use Podman, the containers are launched in `rootless mode
<https://docs.podman.io/en/latest/markdown/podman.1.html#rootless-mode>`__. Your HPC cluster
administrator should have completed most of the configuration for you, but there may be additional
per-user configuration that is required. Before attempting to launch Determined jobs, verify that
you can run simple Podman containers on a compute node. For example:

.. code:: bash

   podman run hello-world

If you are unable to do that successfully, then one or more of the following configuration changes
may be required in your ``$HOME/.config/containers/storage.conf`` file:

#. Podman does not support rootless container storage on distributed file systems (e.g. NFS, Lustre,
   GPSF). On a typical HPC cluster, user directories are on a distributed file system and the
   default container storage location of ``$HOME/.local/share/containers/storage`` is therefore not
   supported. If this is the case on your HPC cluster, configure the ``graphroot`` option in your
   ``storage.conf`` to specify a local file system available on compute nodes. Alternatively, you
   can request that your system administrator configure the ``rootless_storage_path`` in
   ``/etc/containers/storage.conf`` on all compute nodes.

#. Podman utilizes the directory specified by the environment variable ``XDG_RUNTIME_DIR``.
   Normally, this is provided by the login process. Slurm and PBS, however, do not provide this
   variable when launching jobs on compute nodes. When ``XDG_RUNTIME_DIR`` is not defined, Podman
   attempts to create the directory ``/run/user/$UID`` for this purpose. If ``/run/user`` is not
   writable by a non-root user, then Podman commands will fail with a permission error. To avoid
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

#. Unlike Singularity or Podman, you must manually download the Docker image file to the local file
   system (``enroot import``) and then each user must create an Enroot container using that image
   (``enroot create``). When the HPC launcher generates the enroot command for a job, it
   automatically applies the same transformation to the name that Enroot does on import (``/`` and
   ``:`` characters are replaced with ``+``) to enable Docker image references to match the
   associated Enroot container. The following shell commands will download and then create an Enroot
   container for the current user. If other users have read access to ``/shared/enroot/images``,
   they need only perform the ``enroot create`` step to make the container available for their use.

   .. code:: bash

      image=determinedai/environments:cuda-11.3-pytorch-1.12-tf-2.11-gpu-14cb565
      cd /shared/enroot/images
      enroot import docker://$image
      enroot create /shared/enroot/images/${image//[\/:]/\+}.sqsh

#. The Enroot container storage directory for the user ``${ENROOT_CACHE_PATH}`` (which defaults to
   ``$HOME/.local/share/enroot``) must be accessible on all compute nodes.

#. A convenience script, ``/usr/bin/manage-enroot-cache``, is provided by the HPC launcher
   installation to simplify the :ref:`management of enroot images <manage-enroot-cache>`.
