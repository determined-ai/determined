.. _slurm-requirements:

###########################
 Installation Requirements
###########################

********************
 Basic Requirements
********************

Deploying Determined with Slurm/PBS has the following requirements.

-  The login node, admin node, and compute nodes must be configured with Ubuntu 20.04 or later,
   CentOS 7 or later, or SLES 15 or later.

      -  User and group configuration must be consistent across all nodes of the HPC cluster
      -  All nodes must be able to resolve the hostnames of all other nodes in the HPC cluster
      -  A cluster-wide file system with consistent path names across the HPC cluster

-  Slurm 20.02 or greater or PBS 2021.1.2 or greater.

-  Apptainer 1.0 or greater, Singularity 3.7 or greater, Enroot 3.4.0 or greater or PodMan 3.3.1 or
   greater.

-  A cluster-wide shared filesystem.

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

      Given those options, the simplest path to enable Determined job preemption is by including
      ``D`` in the ``preemption_order``. You may include ``R`` in the ``preemption_order``, but it
      is disabled for Determined jobs. You may include ``C`` to the ``preemption_order`` if you
      additionally configure a checkpoint script. Refer to the PBS documentation for details. If you
      choose to implement a checkpoint script, you may initiate a Determined checkpoint by sending a
      ``SIGTERM`` signal to the Determined job. When a Determined job receives a ``SIGTERM``, it
      begins a checkpoint and graceful shutdown. To prevent unnecessary loss of work, it is
      recommended that you wait for at least one Determined ``scheduling_unit`` for the job to
      complete after sending the ``SIGTERM``. If after that period of time the job has not
      terminated, then send a ``SIGKILL`` to forcibly release all resources.

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
   Normally, ``XDG_RUNTIME_DIR`` is set to ``/run/user/$UID`` by the login process. However, Slurm
   and PBS do not provide this variable when launching jobs on compute nodes. Even if
   ``XDG_RUNTIME_DIR`` is not set, podman will automatically attempt to create the
   ``/run/user/$UID`` directory, but will fail with a permission error because ``/run/user`` is not
   writable by a non-root user. Furthermore, if ``XDG_RUNTIME_DIR`` is not set, podman may fail to
   clean up its processes when a job is cancelled, leaving the container in a running or corrupted
   state that may require a ``podman system migrate`` to fix. For Slurm versions older than 22, it
   may also cause Slurm to place the node in the ``drain`` state, thereby preventing other jobs from
   being run on that node.

   To avoid this problem, the System Administrator can create the ``/run/user/$UID`` directory for
   each user on all compute nodes, with permissions 700, and ownership and group set to the username
   and group. The System Administrator must also set the ``XDG_RUNTIME_DIR`` environment variable in
   the ``/etc/determined/master.yaml``, as shown below.

      .. code:: yaml

         task_container_defaults:
            environment_variables:
               - XDG_RUNTIME_DIR=/run/user/$(id -u)

   After modifying ``/etc/determined/master.yaml``, restart the Determined master with
   ``systemctl restart determined-master``.

#. PodMan creates several processes when running a container, such as ``podman``, ``conmon``, and
   ``catatonit``. The ``catatonit`` process forwards signals to the spawned child, tears down the
   container when the spawned child exists, and cleans up exited processes. When Slurm is configured
   to use ``ProctrackType=proctrack/cgroup``, cancelling a job may fail to terminate the processes
   running inside the container, leaving the container in a running or corrupted state that may
   require a ``podman system migrate`` to fix. This is because when a job is cancelled, Slurm will
   send a SIGTERM to all processes that are part of the ``cgroup``, including ``catatonit``, which
   is the processes responsible for tearing down the container. For Slurm versions older than 22, it
   may also cause Slurm to place the node in the ``drain`` state, thereby preventing other jobs from
   being run on that node.

   It should be noted that once the ``catatonit`` process is started, if the container is allowed to
   run until completion without being terminated, the ``catatonit`` process will remain running
   after all other podman related processes have terminated. In this case, running another job that
   starts a podman container, and then cancelling that job, will not cause the problem described
   above to occur, because the existing ``catatonit`` process is not part of the same ``cgroup`` as
   the new podman processes.

   To ensure that the problem described above will never occur, create a Task Epilog script that
   will send a SIGTERM to any process in the cgroup that Slurm did not send a SIGTERM to. This will
   allow those processes that are still running after the job has been cancelled to properly clean
   up and tear down the container before Slurm sends them a final SIGKILL.

   Set the Task Epilog script in the ``slurm.conf`` file, as shown below, to point to a script that
   resides in a shared filesystem that is accessible from all compute nodes.

      .. code::

         TaskEpilog=/path/to/task_epilog.sh

   Set the contents of the Task Epilog script as shown below. Ensure that the ``/sys/fs/cgroup/...``
   path is appropriate for your particular Operating System.

      .. code:: bash

         #!/usr/bin/env bash

         # Send a SIGTERM to any process still active in the cgroup before Slurm sends it a SIGKILL.
         for pid in $(cat /sys/fs/cgroup/freezer/slurm/uid_$(id -u)/job_${SLURM_JOBID}/step_0/cgroup.procs)
         do
            # Check if the process is still active, as it may have already been terminatd as a
            # result of killing one of the other processes.
            if [ -e /proc/${pid} ]
            then
               echo "$(date):$0: Killing $(readlink /proc/${pid}/exe) (${pid})" 1>&2

               kill -SIGTERM ${pid}

               # Wait 15 seconds for the process to terminate to avoid having Slurm send it
               # a SIGKILL before the process gets a chance to clean up.
               timeout -k 15s 15s bash -c "while ps -p ${pid} > /dev/null 2>&1; do sleep 1; done"
            fi
         done

   Restart ``slurmd`` on all the compute nodes after making the change.

.. _enroot-config-requirements:

*********************
 Enroot Requirements
*********************

Install and configure Enroot on all compute nodes of your cluster as per the `Enroot Installation
instructions <https://github.com/NVIDIA/enroot/blob/master/doc/installation.md>`__ for your
platform. There may be additional per-user configuration that is required.

   #. Enroot utilizes the directory ``${ENROOT_RUNTIME_PATH}`` (with default value
      ``${XDG_RUNTIME_DIR}/enroot``) for temporary files. Normally ``XDG_RUNTIME_DIR`` is provided
      by the login process, but Slurm and PBS do not provide this variable when launching jobs on
      compute nodes. When neither ENROOT_RUNTIME_PATH/XDG_RUNTIME_DIR is defined, Enroot attempts to
      create the directory /run/enroot for this purpose. This typically fails with a permission
      error for any non-root user. Select one of the following alternatives to ensure that
      ``XDG_RUNTIME_DIR`` or ``ENROOT_RUNTIME_PATH`` is defined and points to a user-writable
      directory when Slurm/PBS jobs are launched on the cluster.

         -  Have your HPC cluster administrator configure Slurm/PBS to provide ``XDG_RUNTIME_DIR``, or
               change the default ``ENROOT_RUNTIME_PATH`` defined in ``/etc/enroot/enroot.conf`` on
               each node in your HPC cluster.

         -  If using Slurm, provide an ``ENROOT_RUNTIME_PATH`` definition in
            ``task_container_defaults.environment_variables`` in master.yaml.

               .. code:: yaml

                  task_container_defaults:
                     environment_variables:
                        - ENROOT_RUNTIME_PATH=/tmp/$(whoami)

         -  If using Slurm, provide an ``ENROOT_RUNTIME_PATH`` definition in your experiment
            configuration.

   #. Unlike Singularity or PodMan, you must manually download the docker image file to the local
      file system (``enroot import``) and then each user must create an Enroot container using that
      image (``enroot create``). When the HPC launcher generates the enroot command for a job, it
      automatically applies the same transformation to the name that Enroot does on import (``/``
      and ``:`` characters are replaced with ``+``) to enable docker mage references to match the
      associated Enroot container. The following shell commands will download and then create an
      Enroot container for the current user. If other users have read access to
      ``/shared/enroot/images``, they need only perform the ``enroot create`` step to make the
      container available for their use.

         .. code:: bash

            image=determinedai/environments:cuda-11.3-pytorch-1.10-tf-2.8-gpu-24586f0
            cd /shared/enroot/images
            enroot import docker://$image
            enroot create /shared/enroot/images/${image//[\/:]/\+}

   #. The Enroot container storage directory for the user ``${ENROOT_CACHE_PATH}`` (which defaults
      to ``$HOME/.local/share/enroot``) must be accessible on all compute nodes.
