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

-  Slurm 19.05 or greater or PBS 2021.1.2 or greater.

-  Singularity 3.7 or greater or PodMan 3.3.1 or greater.

-  A cluster-wide shared filesystem.

-  To run jobs with GPUs, the Nvidia drivers must be installed on each compute node. Determined
   requires a version greater than or equal to 450.80 of the Nvidia drivers. The Nvidia drivers can
   be installed as part of a CUDA installation but the rest of the CUDA toolkit is not required.

-  Determined supports the `active Python versions <https://endoflife.date/python>`__.

***********************
 Launcher Requirements
***********************

The launcher has the following additional requirements on the installation node:

-  Support for an RPM or Debian-based package installer
-  Java 1.8 or greater
-  Sudo is configured to process configuration files present in the ``/etc/sudoers.d`` directory
-  Access to the Slurm or PBS command line interface for the cluster
-  Access to a cluster-wide file system with a consistent path name across the cluster

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

   -  The Determined default resource pool is set to the Slurm default partition.

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

-  Ensure homogeneous PBS queues.

   Determined maps PBS queues to Determined resource pools. It is recommended that the nodes within
   a queue be homogeneous for Determined to effectively schedule GPU jobs.

   -  A PBS queue with GPUs is identified as a CUDA/ROCM resource pool. The type is inherited from
      the ``resource_manager.slot_type`` configuration. It can be also be specified per partition
      using ``resource_manager.partition_overrides``.

   -  A PBS queue with no GPUs is identified as an AUX resource pool.

   -  The Determined default resource pool is set to the PBS default queue.

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
