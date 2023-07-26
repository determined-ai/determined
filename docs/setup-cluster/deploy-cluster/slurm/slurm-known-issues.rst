.. _known-hpc-issues:

##############
 Known Issues
##############

***********************************************
 Agent-Specific Scheduling Options are Ignored
***********************************************

When using the HPC Launcher, Determined delegates all job scheduling and prioritization to the HPC
workload manager (either Slurm or PBS) and the following experiment configuration options are
ignored.

-  ``resources.agent_label``
-  ``resources.max_slots``
-  ``resources.priority``
-  ``resources.weight``

.. _slurm-and-docker-differences:

************************************
 Singularity and Docker Differences
************************************

Some constraints are due to differences in behavior between Docker and Singularity, summarized here:

-  Singularity tends to explicitly share resources/devices from the host compute node on which it is
   running which results in more opportunities for conflicts with other programs running on the
   cluster, or between multiple determined experiments that are launched concurrently on the same
   compute node.

   -  By default ``/tmp`` and ``/dev/shm`` are mounted from the compute node instead of private to
      the container. If multiple containers are running on the same node there can be more sharing
      than they expect. The contents of ``/tmp`` persist beyond the container lifetime and are
      visible to other trials. The experiment configuration might need to be updated to accommodate
      these issues.

   -  Determined mitigates potential file name and disk space conflicts on ``/tmp`` content by
      automatically using space in ``job_storage_root`` for a per-job ``/tmp`` directory. You can
      override this behavior by providing an explicit bind mount of the ``container_path`` ``/tmp``
      folder in the Singularity container.

   You can restore the default Singularity behavior of sharing ``/tmp`` on the compute node by
   including the following :ref:`bind mount <exp-bind-mounts>` in your experiment configuration or
   globally by using the ``task_container_defaults`` section in your master configuration:

   .. code:: yaml

      bind_mounts:
         - host_path: /tmp
           container_path: /tmp

   -  The ``singularity.conf`` options can also be used to change this behavior, or by using
      individual environment variables added to your experiment. Here are some configuration options
      that might be useful to tune sharing available in the ``singularity.conf`` file:

      +-------------------------+----------------------------------------------------------------+
      | Option                  | Description                                                    |
      +=========================+================================================================+
      | ``sessiondir max size`` | Controls the disk space, in MB, allocated to support           |
      |                         | directories not shared from the host compute node, such as     |
      |                         | ``/tmp`` and ``/usr/tmp``, depending upon your configuration.  |
      +-------------------------+----------------------------------------------------------------+
      | ``mount tmp``           | Isolates ``/tmp`` from the host compute node. The size of this |
      |                         | area is configured by sessiondir max size.                     |
      +-------------------------+----------------------------------------------------------------+

-  Singularity attempts to automatically download and convert Docker images, however, the behavior
   is somewhat different than with Docker.

   -  By default converted Singularity images are stored per user in ``~/.singularity``. Determined
      environment images are relatively large and this can result in excessive duplication.

   -  You likely want to predownload images under ``singularity_image_root`` as described in
      :ref:`slurm-image-config` or configure ``SINGULARITY_CACHEDIR`` to point to a shared
      directory.

-  Some Docker features do not have an exact replacement in Singularity, and therefore the
   associated Determined features are not supported.

   +--------------------------------------+------------------------------------------------------+
   | Feature                              | Description                                          |
   +======================================+======================================================+
   | ``resources.devices``                | By default ``/dev`` is mounted from the compute      |
   |                                      | host, so all devices are available. This can be      |
   |                                      | overridden by the ``singularity.conf`` ``mount dev`` |
   |                                      | option.                                              |
   +--------------------------------------+------------------------------------------------------+
   | ``resources.shm_size``               | By default ``/dev/shm`` is mounted from the compute  |
   |                                      | host. This can be overridden by the                  |
   |                                      | ``singularity.conf`` ``mount tmp`` option. When      |
   |                                      | enabled, the size can be increased using compute     |
   |                                      | node ``/etc/fstab`` settings.                        |
   +--------------------------------------+------------------------------------------------------+
   | ``environment.registry_auth.server`` | No equivalent setting in Singularity.                |
   +--------------------------------------+------------------------------------------------------+
   | ``environment.registry_auth.email``  | No equivalent setting in Singularity.                |
   +--------------------------------------+------------------------------------------------------+

**************************
 Singularity Known Issues
**************************

Launching a PBS job with an experiment configuration that includes an embedded double quote
character (") may cause the job to fail unless you have Singularity 3.10 or greater or Apptainer 1.1
or greater. For example, the error might be the json.decoder.JSONDecodeError or the experiment log
may contain ``source: /.inject-singularity-env.sh:224:1563: "export" must be followed by names or
assignments`` and ``RuntimeError: missing environment keys [DET_MASTER, DET_CLUSTER_ID,
DET_AGENT_ID, DET_SLOT_IDS, DET_TASK_ID, DET_ALLOCATION_ID, DET_SESSION_TOKEN, DET_TASK_TYPE], is
this running on-cluster?``

The version of Singularity is detected by the HPC Launcher invoking the singularity command and
checking for the ``--no-eval`` option. If the singularity command is not on the path for the HPC
launcher or is of an inconsistent version with the compute nodes, embedded double quote characters
may still not work.

************************
 Apptainer Known Issues
************************

Starting with Apptainer version 1.1.0 some changes may trigger permission problems inside of
Determined containers for shells, tensorboards, and experiments. For example, a tensorboard log may
contain ``ERROR: Could not install packages due to an OSError: [Errno 28] No space left on device``,
or a shell may fail to function and the shell logs contain the message ``chown(/dev/pts/1, 63200, 5)
failed: Invalid argument``, or an experiment may fail to launch due to ``FATAL: container creation
failed: mount /var/tmp->/var/tmp error: while mounting /var/tmp: could not mount /var/tmp: operation
not supported``. This likely indicates an installation or configuration error for unprivileged
containers. Review the `Installing Apptainer
<https://apptainer.org/docs/admin/main/installation.html>`_ documentation. These errors are
sometimes resolved by additionally installing the ``apptainer-setuid`` package.

*********************
 Podman Known Issues
*********************

-  Determined uses Podman in `rootless mode
   <https://docs.podman.io/en/latest/markdown/podman.1.html#rootless-mode>`__. There are several
   configuration errors that may be encountered:

   -  ``stat /run/user/NNN: no such file or directory`` likely indicates that the environment
      variable ``XDG_RUNTIME_DIR`` is referencing a directory that does not exist.

   -  ``stat /run/user/NNN: permission denied`` may indicate a problem with default the ``runroot``
      configuration.

   -  ``Error: A network file system with user namespaces is not supported. Please use a
      mount_program: backing file system is unsupported for this graph driver`` indicates that the
      ``graphroot`` references a distributed file system.

   Refer to :ref:`podman-config-requirements` for recommendations.

-  On a Slurm cluster, it is common to rely upon ``/etc/hosts`` (instead of DNS) to resolve the
   addresses of the login node and other compute nodes in the cluster. If jobs are unable to resolve
   the address of the Determined master or other compute nodes in the job and you are relying on
   ``/etc/hosts``, check the following:

   #. Ensure that the ``/etc/hosts`` file is being mounted in the container by a :ref:`bind mount
      <exp-bind-mounts>` in the ``task_container_defaults`` section of your master configuration as
      shown below. Unlike Singularity, Podman V4.0+ no longer maps ``/etc/hosts`` from the host into
      the running container by default. On the initial startup, the Determined Slurm launcher
      automatically adds the ``task_container_defaults`` fragment below when adding the
      ``resource_manager`` section. If, however, you have since changed the file you may need to
      manually add the :ref:`bind mount <exp-bind-mounts>` to ensure that jobs can resolve all host
      addresses in the cluster:

      .. code:: yaml

         task_container_defaults:
            bind_mounts:
               -  host_path: /etc/hosts
                  container_path: /etc/hosts

   #. Ensure that the names and addresses of the login node, admin node, and all compute nodes are
      consistently available in ``/etc/hosts`` on all nodes.

-  Podman containers only inherit environment variables that have been explicitly specified.
   Determined adds Podman arguments to provide any Determined-configured environment variables, and
   the launcher enables inheritance of the following variables: ``SLURM_*``,
   ``CUDA_VISIBLE_DEVICES``, ``NVIDIA_VISIBLE_DEVICES``, ``ROCR_VISIBLE_DEVICES``,
   ``HIP_VISIBLE_DEVICES``. You may enable the inheritance of additional variables from the host
   environment by specifying the variable name with an empty value in the ``environment_variables``
   of your experiment configuration or :ref:`task container defaults
   <master-task-container-defaults>`.

   .. code:: yaml

      environment_variables:
         - INHERITED_ENV_VAR=

-  Terminating a Determined AI job may cause the following conditions to occur:

   -  Compute nodes go into drain state.

   -  Processes inside the container continue to run.

   -  An attempt to run another job results in ``Running a job gets the error level=error
      msg="invalid internal status, try resetting the pause process with \"/usr/local/bin/podman
      system migrate\": could not find any running process: no such process"``.

   Podman creates several processes when running a container, such as podman, conmon, and catatonit.
   When a user terminates a Determined AI job, Slurm will send a SIGTERM to the podman processes.
   However, sometimes the container will continue running, even after the SIGTERM has been sent.

   On Slurm versions prior to version 22, Slurm will place the node in the ``drain`` state,
   requiring the use of the ``scontrol`` command to set the node back to the ``idle`` state. It may
   also require ``podman system migrate`` to be run to clean up the running containers.

   To ensure the container associated with the job is stopped when a Determined AI job is
   terminated, create a Slurm task epilog script to stop the container.

   Set the Task Epilog script in the ``slurm.conf`` file, as shown below, to point to a script that
   resides in a shared filesystem accessible from all compute nodes.

   .. code::

      TaskEpilog=/path/to/task_epilog.sh

   Set the contents of the Task Epilog script as shown below.

   .. code:: bash

      #!/usr/bin/env bash

      slurm_job_name_suffix=$(echo ${SLURM_JOB_NAME} | sed 's/^\S\+-\([a-z0-9]\+-[a-z0-9]\+\)$/\1/')

      if ps -fe | grep -E "[p]odman run .*-name ${SLURM_JOB_USER}-\S+-${slurm_job_name_suffix}" > /dev/null
      then
         timeout -k 15s 15s bash -c "while ps -fe | grep -E \"[c]onmon .*-n ${SLURM_JOB_USER}-\S+-${slurm_job_name_suffix}\" > /dev/null 2>&1; do sleep 1; done"

         podman_container_stop_command="podman container stop --filter name='.+-${slurm_job_name_suffix}'"

         echo "$(date):$0: Running \"${podman_container_stop_command}\"" 1>&2

         eval ${podman_container_stop_command}
      fi

      exit 0

   Restart the ``slurmd`` daemon on all compute nodes.

*********************
 Enroot Known Issues
*********************

-  Enroot uses ``XDG_RUNTIME_DIR`` which is not provided to the compute jobs by Slurm/PBS by
   default. The error ``mkdir: cannot create directory ‘/run/enroot’: Permission denied`` indicates
   that the environment variable ``XDG_RUNTIME_DIR`` is not defined on the compute nodes. See
   :ref:`podman-config-requirements` for recommendations.

-  Enroot requires manual download and creation of containers. The error ``[ERROR] No such file or
   directory:
   /home/users/test/.local/share/enroot/determinedai+environments+cuda-11.1-base-gpu-mpi-0.18.5``
   indicates the user ``test`` has not created an Enroot container for Docker image
   ``determinedai/environments:cuda-11.1-base-gpu-mpi-0.18.5``. Check the available containers using
   the ``enroot list`` command. See :ref:`enroot-config-requirements` for guidance on creating
   Enroot containers.

-  Enroot does not provide a mechanism for sharing containers. Each user must create any containers
   needed by their Determined experiments prior to creating the experiment.

-  Some Docker features do not have an exact replacement in Enroot, and therefore the associated
   Determined features are not supported.

   +--------------------------------------+------------------------------------------------------+
   | Feature                              | Description                                          |
   +======================================+======================================================+
   | ``resources.devices``                | Managed via Enroot configuration files.              |
   +--------------------------------------+------------------------------------------------------+
   | ``resources.shm_size``               | Managed via Enroot configuration files.              |
   +--------------------------------------+------------------------------------------------------+
   | ``environment.registry_auth.server`` | No equivalent setting in Enroot.                     |
   +--------------------------------------+------------------------------------------------------+
   | ``environment.registry_auth.email``  | No equivalent setting in Enroot.                     |
   +--------------------------------------+------------------------------------------------------+

.. _slurm-known-issues:

********************
 Slurm Known Issues
********************

-  Jobs may fail to submit with Slurm version 22.05.5 through 22.05.8 with the message ``error:
   Unable to allocate resources: Requested node configuration is not available``.

   Slurm 22.05.5 through 22.05.8 are not supported due to `Slurm Bug 15857
   <https://bugs.schedmd.com/show_bug.cgi?id=15857>`__. The bug was addressed in 22.05.09 or
   23.02.00.

-  A Determined experiment remains ``QUEUEUED`` for an extended period:

   If Slurm provides a reason code for the ``QUEUEUED`` state of the job, the reason description
   from `JOB REASON CODES <https://slurm.schedmd.com/squeue.html#SECTION_JOB-REASON-CODES>`__ will
   be added to the experiment/task log as an informational message such as:

   .. code::

      INFO: HPC job waiting to be scheduled: Nodes required for job are DOWN, DRAINED or reserved for jobs in higher priority partitions

   In some cases, it may be helpful to inspect the details of your queued jobs using the Slurm
   ``scontrol show jobs`` command using the ``HPC Job ID`` displayed in the experiment/task log. An
   example of the command output is shown below.

   .. code::

      $ scontrol show job 109084
      JobId=109084 JobName=det-ai_exp-2221-trial-15853-2221.33b6fcca-564d-47a7-ab2e-0d2a4a90a0f1.1
      UserId=user(1234) GroupId=users(100) MCS_label=N/A
      Priority=4294866349 Nice=0 Account=(null) QOS=normal
      JobState=PENDING Reason=Priority Dependency=(null)
      Requeue=0 Restarts=0 BatchFlag=1 Reboot=0 ExitCode=0:0
      RunTime=00:00:00 TimeLimit=1-00:00:00 TimeMin=N/A
      SubmitTime=2023-07-03T16:01:35 EligibleTime=2023-07-03T16:01:35
      AccrueTime=2023-07-03T16:01:35
      StartTime=Unknown EndTime=Unknown Deadline=N/A
      SuspendTime=None SecsPreSuspend=0 LastSchedEval=2023-07-03T16:06:15 Scheduler=Backfill:*
      Partition=mlde_rocm AllocNode:Sid=o184i054:755599
      ReqNodeList=o186i[122-123] ExcNodeList=(null)
      NodeList=
      NumNodes=1-1 NumCPUs=1 NumTasks=1 CPUs/Task=1 ReqB:S:C:T=0:0:*:*
      ReqTRES=cpu=1,mem=256G,node=1,billing=1,gres/gpu=1
      AllocTRES=(null)
      Socks/Node=* NtasksPerN:B:S:C=1:0:*:* CoreSpec=*
      MinCPUsNode=1 MinMemoryNode=0 MinTmpDiskNode=0
      Features=(null) DelayBoot=00:00:00
      OverSubscribe=OK Contiguous=0 Licenses=(null) Network=(null)
      Command=/cstor/determined/o184i054-jobs/jobs/environments/vishnu/2221.33b6fcca-564d-47a7-ab2e-0d2a4a90a0f1.1/ai_exp-2221-trial-15853-job.sh
      WorkDir=/var/tmp
      StdErr=/cstor/determined/o184i054-jobs/jobs/environments/vishnu/2221.33b6fcca-564d-47a7-ab2e-0d2a4a90a0f1.1/ai_exp-2221-trial-15853-error.log
      StdIn=/dev/null
      StdOut=/cstor/determined/o184i054-jobs/jobs/environments/vishnu/2221.33b6fcca-564d-47a7-ab2e-0d2a4a90a0f1.1/ai_exp-2221-trial-15853-output.log
      Power=
      CpusPerTres=gres:gpu:64
      MemPerTres=gres:gpu:262144
      TresPerJob=gres:gpu:1

   The Slurm job state (See `JOB STATE CODES
   <https://slurm.schedmd.com/squeue.html#SECTION_JOB-STATE-CODES>`__) may help identify the delay
   in scheduling. If the Slurm job state is ``PENDING``, review the resources being requested and
   the ``Reason`` code to identify the cause. To better understand how resource requests are derived
   by Determined, see :ref:`hpc_launching_architecture`. Some common reason codes for ``PENDING``
   are:

   -  ``PartitionNodeLimit``: Ensure that the job is not requesting more nodes than ``MaxNodes`` of
      the partition.

      Ensure that the ``MaxNodes`` setting for the partition is at least as high as the number of
      GPUs in the partition. The ``MaxNodes`` value for a partition can be viewed in the
      ``JOBS_SIZE`` column of the command:

      .. code:: bash

         sinfo -O Partition,Size,Gres,OverSubscribe,NodeList,StateComplete,Reason
         PARTITION  JOB_SIZE    GRES         OVERSUBSCRIBE NODELIST STATECOMPLETE REASON
         defq*      1-infinite  gpu:tesla:4  NO            node002  idle          none

      Until scheduled, the job's ``NumNodes`` is shown as the range 1-``slots_per_trial``. Ensure
      the ``slots_per_trial`` shown is not larger than the value shown in the ``JOB_SIZE`` column
      for the partition.

      A second potential cause of ``PartitionNodeLimit`` is submitting CPU experiments (or when the
      Determined cluster is configured with ``gres_supported: false`` ), without specifying
      ``slurm.slots_per_node`` to enable multiple CPUs to be used on each node. Without
      ``slurm.slots_per_node`` the job will request ``slots_per_trial`` nodes.

   -  ``Priority``: One or more higher priority jobs exist for this partition or advanced
      reservation.

   -  ``Resources``: Expected when resources are in use by other jobs. Otherwise, verify you have
      not requested more resources (GPUs, CPUs, nodes, memory) than are available in your cluster.

.. _pbs-known-issues:

******************
 PBS Known Issues
******************

-  When ``job_history_enable = False``, the following behavior is observed

   -  Jobs are treated as succesful even in the presence of a failure as the job cannot be accessed
      after completion. Further, jobs will show state ``COMPLETED`` in the UI even if an error
      causes

   -  Jobs are not restarted on failure since they are assumed successful.

***********************
 AMD/ROCm Known Issues
***********************

-  AMD/ROCm support is available only with Singularity containers. While Determined does add the
   proper Podman arguments to enable ROCm GPU support, the capabilities have not yet been verified.

-  Launching experiments with ``slot_type: rocm``, may fail with the error ``RuntimeError: No HIP
   GPUs are available``. Ensure that the compute nodes are providing ROCm drivers and libraries
   compatible with the environment image that you are using and that they are available in the
   default locations, or are added to the ``path`` and/or ``ld_library_path`` variables in the
   :ref:`slurm configuration <cluster-configuration-slurm>`. Depending upon your system
   configuration, you may need to select a different ROCm image. See
   :doc:`/model-dev-guide/prepare-container/set-environment-images` for the images available.

-  Launching experiments with ``slot_type: rocm``, may fail in the AMD/ROCm libraries with with the
   error ``terminate called after throwing an instance of 'boost::filesystem::filesystem_error'
   what(): boost::filesystem::remove: Directory not empty: "/tmp/miopen-...``. A potential
   workaround is to disable the per-container ``/tmp`` by adding the following :ref:`bind mount
   <exp-bind-mounts>` in your experiment configuration or globally by using the
   ``task_container_defaults`` section in your master configuration:

   .. code:: yaml

      bind_mounts:
         - host_path: /tmp
           container_path: /tmp

***************************************
 Determined AI Experiment Requirements
***************************************

Ensure that the following requirements are met in your experiment configuration.

Distributed jobs *must* allocate the same number of resources on each compute node. Slurm/PBS will
not enforce this constraint by default. It is, therefore, recommended that you include a
``slots_per_node`` in your experiment configuration to ensure that Slurm/PBS provides a consistent
allocation on each node. Your ``slots_per_trial`` configuration should then be a multiple of
``slots_per_node``.

*************************
 Additional Known issues
*************************

-  The Determined master may fail to show HPC cluster information and report ``Failed to communicate
   with launcher due to error:`` in the ``Master Logs`` tab of the Determined UI. If so, verify the
   following:

   #. Ensure that the launcher service is up and running.

      .. code:: bash

         sudo systemctl status launcher

   #. If the full error is ``Failed to communicate with launcher due to error: {401 Unauthorized}``,
      the Determined master does not have an up-to-date authorization token to access the launcher.
      Restart the launcher, to ensure all configuration changes have been applied.

      .. code:: bash

         sudo systemctl restart launcher
         sudo systemctl status launcher

      Once it has successfully started, you should see the message ``INFO: launcher server ready
      ...``, then restart the Determined master so it will likewise load the latest configuration:

      .. code:: bash

         sudo systemctl restart determined-master
         sudo systemctl status determined-master

      Additional diagnostic messages may be present in the system log diagnostics, such as
      ``/var/log/messages`` or ``journalctl --since=yesterday -u launcher``, and ``journalctl
      --since=yesterday -u determined-master``

-  The SSH server process within Determined Environment images can fail with a ``free(): double free
   detected in tcache 2`` message, a ``Fatal error: glibc detected an invalid stdio handle``
   message, or simply close the connection with no message. This problem has been observed when
   using the ``det shell start`` command and when running distributed, multi-node, training jobs. It
   is suspected to be triggered by passwd/group configurations that use NIS/YP/LDAP accounts on the
   compute host. By default these settings are propagated to the Singularity container and can
   result in ``sshd`` aborting the connection with or without an error message, depending on the
   exact configuration.

   A workaround is to specify a customized ``nsswitch.conf`` file to the Singularity container and
   enable only files for passwd/group elements. This can be accomplished using the following steps:

   #. Create a file on a shared file system such as ``/home/shared/determined/nsswitch.conf`` file
      with the content, potentially further tuned for your environment:

      .. code:: yaml

         passwd: files determined
         shadow: files determined
         group: files determined
         hosts: files dns

   #. Update the Determined cluster configuration to supply a default bind mount to override the
      ``/etc/nsswitch.conf`` in the container.

      .. code:: yaml

         task_container_defaults:
           bind_mounts:
             - host_path: /home/shared/determined/nsswitch.conf
               container_path: /etc/nsswitch.conf

   #. Reload the Determined master to allow it to pull in the updated configuration.

   The user/group configuration is typically injected in ``/etc/passwd`` within the Singularity
   container so disabling the NIS/YP/LDAP accounts within the container should not result in any
   lost capability.

-  Determined CLI can fail with a ``Your requested host "localhost" could not be resolved by DNS.``
   message. This has been observed when the ``http_proxy`` or ``https_proxy`` environment variables
   are set but have not excluded sending ``localhost``, or the Determined master hostname, to the
   proxy server.

   Update the environment settings configured for the proxy to also include:

   .. code:: bash

      export no_proxy=localhost,127.0.0.1

-  The automated download of Docker containers by Singularity may fail with the error ``loading
   registries configuration: reading registries.conf.d: lstat
   /root/.config/containers/registries.conf.d: permission denied`` when Docker login information is
   not provided.

   This happens when access to an otherwise public container image is being blocked by the `Docker
   Hub download rate limit <https://docs.docker.com/docker-hub/download-rate-limit>`__, or if the
   container is in a private registry.

   You can avoid this problem by either:

   #. Manually downloading the container image as described in :ref:`slurm-image-config`.
   #. Providing a Docker login via the experiment configuration using the
      ``environment.registry_auth.username`` and ``environment.registry_auth.password`` options.

-  Use of `NVIDIA Multi-Process Service (MPS) <https://docs.nvidia.com/deploy/mps>`__ with
   Determined may trigger the error ``RuntimeError: CUDA error: all CUDA-capable devices are busy or
   unavailable``.

   By default, MPS depends upon a shared ``/tmp`` directory between the compute node and the
   container to function properly. As noted in :ref:`slurm-and-docker-differences`, sharing ``/tmp``
   between the compute node and the container is not the default behavior for Determined Slurm
   integration. When using MPS, use one of the following workarounds:

   #. If the capabilities of MPS are not required, disable or uninstall the MPS service. See
      `nvidia-cuda-mps-control <https://docs.nvidia.com/deploy/mps/index.html#topic_5_1_1>`__ or the
      relevant documentation associated with your installation package.

   #. Configure the MPS variable ``CUDA_MPS_PIPE_DIRECTORY`` to use a directory other than ``/tmp``
      (e.g. ``/dev/shm``).

   #. Restore the sharing of ``/tmp`` between the compute node and the container as described in
      :ref:`slurm-and-docker-differences`.

   For more information on MPS, refer to the `NVIDIA Multi-Process Service (MPS) Documentation
   <https://docs.nvidia.com/deploy/mps>`__.

-  Experiments on CPU-only clusters will fail when the requested slot count exceeds the maximum
   number of CPUs on any single node. This behavior is due to a limitation of the Slurm workload
   manager. Slurm does not provide an option to request a certain number of CPUs without specifying
   the number of nodes/tasks. To overcome this limitation of Slurm, Determined will set a default
   value of 1 for the number of nodes. With this workaround, when the users launch an experiment on
   a CPU-only cluster, Slurm tries to identify a single node that can completely satisfy the
   requested number of slots (CPUs). If such a node is available, Slurm will allocate the resources
   and continue the execution of the experiment. Otherwise, Slurm will error stating the resource
   request could not be satisfied, as shown in the below example.

   .. code:: bash

      ERROR: task failed without an associated exit code: sbatch: error: CPU count per node can not
      be satisfied sbatch: error: Batch job submission failed: Requested node configuration is not
      available.

-  A job may fail with the message ``resources failed with non-zero exit code``, Determined reports
   the exit code in the experiment logs. For example, the experiment logs contain ``srun: error:
   node002: task 0: Exited with exit code 7``.

-  The ``det slot enable`` and ``det slot disable`` commands are not supported. Use of these
   commands will print an error message.

-  ``det slot list`` will not display the name of any active Determined tasks.
