.. _master-config-reference:

################################
 Master Configuration Reference
################################

The Determined master supports a range of configuration settings that can be set via a :ref:`YAML
configuration file <topic-guides_yaml>`, environment variables, or command-line options. The
configuration file is normally located at ``/etc/determined/master.yaml`` on the master and is read
when the master starts.

The configuration of an active master can be examined using the Determined CLI with the command
``det master config``.

The master supports the following configuration settings:

-  ``config_file``: Path to the master configuration file. Normally this should only be set via an
   environment variable or command-line option. Defaults to ``/etc/determined/master.yaml``.
-  ``port``: The TCP port on which the master accepts incoming connections. If TLS has been enabled,
   defaults to ``8443``; otherwise defaults to ``8080``.

.. _master-task-container-defaults:

-  ``task_container_defaults``: Specifies defaults for all task containers. A task represents a
   single schedulable unit, such as a trial, command, or tensorboard.

   -  ``shm_size_bytes``: The size (in bytes) of ``/dev/shm`` for Determined task containers.
      Defaults to ``4294967296``.

   -  ``network_mode``: The Docker network to use for the Determined task containers. If this is set
      to ``host``, `Docker host-mode networking <https://docs.docker.com/network/host/>`__ will be
      used instead. Defaults to ``bridge``.

   -  ``dtrain_network_interface``: The network interface to use during :ref:`multi-gpu-training`.
      If not set, Determined automatically determines the network interface to use.

      When training a model with multiple machines, the host network interface used by each machine
      must have the same interface name across machines. The network interface to use can be
      determined automatically, but there may be issues if there is an interface name common to all
      machines but it is not routable between machines. Determined already filters out common
      interfaces like ``lo`` and ``docker0``, but agent machines may have others. If interface
      detection is not finding the appropriate interface, the ``dtrain_network_interface`` option
      can be used to set it explicitly (e.g., ``eth11``).

   -  ``cpu_pod_spec``: Defines the default pod spec which will be applied to all CPU-only tasks
      when running on Kubernetes. See :ref:`custom-pod-specs` for details.

   -  ``gpu_pod_spec``: Defines the default pod spec which will be applied to all GPU tasks when
      running on Kubernetes. See :ref:`custom-pod-specs` for details.

   -  ``image``: Defines the default docker image to use when executing the workload. If a docker
      image is specified in the :ref:`experiment config <exp-environment-image>` this default is
      overriden. This image must be accessible via ``docker pull`` to every Determined agent machine
      in the cluster. Users can configure different container images for NVIDIA GPU tasks using the
      ``cuda`` key (``gpu`` prior to 0.17.6), CPU tasks using ``cpu`` key, and ROCm (AMD GPU) tasks
      using the ``rocm`` key. Default values:

      -  ``determinedai/environments-dev:cuda-11.3-pytorch-1.12-tf-2.8-gpu-0.21.2`` for NVIDIA GPUs.
      -  ``determinedai/environments-dev:rocm-5.0-pytorch-1.10-tf-2.7-rocm-0.21.2`` for ROCm.
      -  ``determinedai/environments-dev:py-3.8-pytorch-1.12-tf-2.8-cpu-0.21.2`` for CPUs.

   -  ``environment_variables``: A list of environment variables that will be set in every task
      container. Each element of the list should be a string of the form ``NAME=VALUE``. See
      :ref:`environment-variables` for more details. Environment variables specified in experiment
      config will override default values specified here. Users can customize environment variables
      for CUDA (NVIDIA GPU), CPU, and ROCm (AMD GPU) tasks differently by specifying a dict with
      ``cuda`` (``gpu`` prior to 0.17.6), ``cpu``, and ``rocm`` keys.

   -  ``force_pull_image``: Defines the default policy for forcibly pulling images from the docker
      registry and bypassing the docker cache. If a pull policy is specified in the :ref:`experiment
      config <exp-environment-image>` this default value is overriden. Please note that as of
      November 1st, 2020 unauthenticated users will be `capped at 100 pulls from Docker per 6 hours
      <https://www.docker.com/blog/scaling-docker-to-serve-millions-more-developers-network-egress/>`__.
      Defaults to ``false``.

   -  ``registry_auth``: Defines the default `docker registry credentials
      <https://docs.docker.com/engine/api/v1.30/#operation/SystemAuth>`__ to use when pulling a
      custom base docker image, if needed. If credentials are specified is in the :ref:`experiment
      config <exp-environment-image>` this default value is overriden. Credentials are specified as
      the following nested fields:

      -  ``username`` (required)
      -  ``password`` (required)
      -  ``serveraddress`` (required)
      -  ``email`` (optional)

   -  ``add_capabilities``: The default list of Linux capabilities to grant to task containers.
      Ignored by resource managers of type ``kubernetes``. See :ref:`environment.add_capabilities
      <exp-environment-add-capapbilities>` for more details.

   -  ``drop_capabilities``: Just like ``add_capabilities`` but for dropping capabilities.

   -  ``devices``: The default list of devices to pass to the Docker daemon. Ignored by resource
      managers of type ``kubernetes``. See :ref:`resources.devices <exp-resources-devices>` for more
      details.

   -  ``bind_mounts``: The default bind mounts to pass to the Docker container. Ignored by resource
      managers of type ``kubernetes``. See :ref:`resources.devices <exp-bind-mounts>` for more
      details.

   -  ``slurm``: Additional Slurm options when launching trials with ``sbatch``. See
      :ref:`environment.slurm <slurm-config>` for more details.

   -  ``pbs``: Additional PBS options when launching trials with ``qsub``. See :ref:`environment.pbs
      <pbs-config>` for more details.

-  ``root``: Specifies the root directory of the state files. Defaults to
   ``/usr/share/determined/master``.

-  ``cache``: Configuration for file cache.

   -  ``cache_dir``: Specifies the root directory for file cache. Defaults to
      ``/var/cache/determined``. Note that the master would break on startup if it does not have
      access to create this default directory.

-  ``cluster_name`` (optional): Specify a human readable name for this cluster.

-  ``tensorboard_timeout``: Specifies the duration in seconds before idle TensorBoard instances are
   automatically terminated. A TensorBoard instance is considered to be idle if it does not receive
   any HTTP traffic. The default timeout is ``300`` (5 minutes).

.. _master-config-notebook-timeout:

-  ``notebook_timeout``: Specifies the duration in seconds before idle notebook instances are
   automatically terminated. A notebook instance is considered to be idle if it is not receiving any
   HTTP traffic and it is not otherwise active (as defined by the ``notebook_idle_type`` option in
   the :ref:`task configuration <command-notebook-configuration>`). Defaults to ``null``, i.e.
   disabled.

-  ``resource_manager``: The resource manager to use to acquire resources. Defaults to ``agent``.

   -  ``type: agent``: The agent resource manager includes static and dynamic agents.

      -  ``scheduler``: Specifies how Determined schedules tasks to agents on resource pools. If a
         resource pool is specified with an individual scheduler configuration, that will override
         the default scheduling behavior specified here. For more on scheduling behavior in
         Determined, see :ref:`scheduling`.

         -  ``type``: The scheduling policy to use when allocating resources between different tasks
            (experiments, notebooks, etc.). Defaults to ``fair_share``.

            -  ``fair_share``: Tasks receive a proportional amount of the available resources
               depending on the resource they require and their weight.

            -  ``round_robin``: Tasks are scheduled in the order which they arrive at the cluster.

            -  ``priority``: Tasks are scheduled based on their priority, which can range from the
               values 1 to 99 inclusive. Lower priority numbers indicate higher priority tasks. A
               lower priority task will never be scheduled while a higher priority task is pending.
               Zero-slot tasks (e.g., CPU-only notebooks, tensorboards) are prioritized separately
               from tasks requiring slots (e.g., experiments running on GPUs). Task priority can be
               assigned using the ``resources.priority`` field. If a task does not specify a
               priority it is assigned the ``default_priority``.

               -  ``preemption``: Specifies whether lower priority tasks should be preempted to
                  schedule higher priority tasks. Tasks are preempted in order of lowest priority
                  first.

               -  ``default_priority``: The priority that is assigned to tasks that do not specify a
                  priority. Can be configured to 1 to 99 inclusively. Defaults to ``42``.

         -  ``fitting_policy``: The scheduling policy to use when assigning tasks to agents in the
            cluster. Defaults to ``best``.

            -  ``best``: The best-fit policy ensures that tasks will be preferentially "packed"
               together on the smallest number of agents.
            -  ``worst``: The worst-fit policy ensures that tasks will be placed on under-utilized
               agents.

         -  ``allow_heterogeneous_fits``: Fit distributed jobs to onto agents of different sizes.
            When enabled, we still prefer to fit jobs on same sized nodes but will fallback to allow
            heterogeneous fits. Sizes should be powers of two for the fitting algorithm to work.

      -  ``default_aux_resource_pool``: The default resource pool to use for tasks that do not need
         dedicated compute resources, auxiliary, or systems tasks. Defaults to ``default`` if no
         resource pool is specified.

      -  ``default_compute_resource_pool``: The default resource pool to use for tasks that require
         compute resources, e.g. GPUs or dedicated CPUs. Defaults to ``default`` if no resource pool
         is specified.

      -  ``require_authentication``: Whether to require that agent connections be verified using
         mutual TLS.

      -  ``client_ca``: Certificate authority file to use for verifying agent certificates.

   -  ``type: kubernetes``: The ``kubernetes`` resource manager launches tasks on a Kubernetes
      cluster. The Determined master must be running within the Kubernetes cluster. When using the
      ``kubernetes`` resource manager, we recommend deploying Determined using the :ref:`Determined
      Helm Chart <install-on-kubernetes>`. When installed via Helm, the configuration settings below
      will be set automatically. For more information on using Determined with Kubernetes, see the
      :ref:`documentation <determined-on-kubernetes>`.

      -  ``namespace``: The namespace where Determined will deploy Pods and ConfigMaps.

      -  ``max_slots_per_pod``: Each multi-slot (distributed training) task will be scheduled as a
         set of ``slots_per_task / max_slots_per_pod`` separate pods, with each pod assigned up to
         ``max_slots_per_pod`` slots. Distributed tasks with sizes that are not divisible by
         ``max_slots_per_pod`` are never scheduled. If you have a cluster of different size nodes,
         set ``max_slots_per_pod`` to the greatest common divisor of all the sizes. For example, if
         you have some nodes with 4 GPUs and other nodes with 8 GPUs, set ``maxSlotsPerPod`` to
         ``4`` so that all distributed experiments will launch with 4 GPUs per pod (with two pods on
         8-GPU nodes).

      -  ``slot_type``: Resource type used for compute tasks. Defaults to ``cuda``.

         -  ``slot_type: cuda``: One NVIDIA GPU will be requested per compute slot. Prior to 0.17.6,
            this option was called ``gpu``.

         -  ``slot_type: cpu``: CPU resources will be requested for each compute slot.
            ``slot_resource_requests.cpu`` option is required to specify the specific amount of the
            resources.

      -  ``slot_resource_requests``: Supports customizing the resource requests made when scheduling
         Kubernetes pods.

         -  ``cpu``: The number of Kubernetes CPUs to request per compute slot.

      -  ``master_service_name``: The service account Determined uses to interact with the
         Kubernetes API.

      -  ``fluent``: Options for configuring how Fluent Bit sidecars are run.

         -  ``image``: The Fluent Bit image to use. Defaults to ``fluent/fluent-bit:1.9.3``.

         -  ``uid``/``gid``: The UID and GID to run the Fluent Bit sidecar as. If these are not
            specified, the container will run as root when the associated task container is running
            as root and as a default non-root user otherwise.

.. _cluster-configuration-slurm:

   -  ``type: slurm`` or ``pbs``: The HPC launcher submits tasks to a Slurm/PBS cluster. For more
      information, see :ref:`using_slurm`.

      -  ``master_host``: The hostname for the Determined master by which tasks will communicate
         with its API server.

      -  ``master_port``: The port for the Determined master.

      -  ``host``: The hostname for the Launcher, which Determined communicates with to launch and
         monitor jobs.

      -  ``port``: The port for the Launcher.

      -  ``protocol``: The protocol for communicating with the Launcher.

      -  ``security``: Security-related configuration settings for communicating with the Launcher.

         -  ``tls``: TLS-related configuration settings.

            -  ``enabled``: Enable TLS.

            -  ``skip_verify``: Skip server certificate verification.

            -  ``certificate``: Path to a file containing the cluster's TLS certificate. Only needed
               if the certificate is not signed by a well-known CA; cannot be specified if
               ``skip_verify`` is enabled.

      -  ``container_run_type``: The type of the container runtime to be used when launching tasks.
         The value may be ``singularity``, ``enroot``, or ``podman``. The default value is
         ``singularity``. The value ``singularity`` is also used when using Apptainer.

      -  ``auth_file``: The location of a file which contains an authorization token to communicate
         with the launcher. It is automatically updated by the launcher as needed when the launcher
         is started. The specified path must be writable by the launcher, and readable by the
         Determined master.

      -  ``slot_type``: The default slot type assumed when users request resources from Determined
         in terms of ``slots``. Available values are ``cuda``, ``rocm`` and ``cpu``, where 1
         ``cuda`` or ``rocm`` slot is 1 GPU. Otherwise, CPU slots are requested. The number of CPUs
         allocated per node is 1, unless overridden by ``slots_per_node`` in the experiment
         configuration. Defaults per-partition to ``cuda`` if GPU resources are found within the
         partition, else ``cpu``. If GPUs cannot be detected automatically, for example when
         operating with ``gres_supported: false``, then this result may be overridden using
         ``partition_overrides``.

         -  ``slot_type: cuda``: One NVIDIA GPU will be requested per compute slot. Partitions will
            be represented as a resource pool with slot type ``cuda`` which can be overridden using
            ``partition_overrides``.

         -  ``slot_type: rocm``: One AMD GPU will be requested per compute slot. Partitions will be
            represented as a resource pool with slot type ``rocm`` which can be overridden using
            ``partition_overrides``.

         -  ``slot_type: cpu``: CPU resources will be requested for each compute slot. Partitions
            will be represented as a resource pool with slot type ``cpu``. One node will be
            allocated per slot.

      -  ``rendezvous_network_interface``: The interface used to bootstrap communication between
         distributed jobs. For example, when using horovod the IP address for the host on this
         interface is passed in the host list to ``horovodrun``. Defaults to any interface beginning
         with ``eth`` if one exists, otherwise the IPv4 resolution of the hostname.

      -  ``proxy_network_interface``: The interface used to proxy the master for services running on
         compute nodes. The interface Defaults to the IPv4 resolution of the hostname.

      -  ``user_name``: The username that the Launcher will run as. It is recommended to set this to
         something other than ``root``. The user must have a home directory with read permissions
         for all users to enable access to generated ``sbatch`` scripts and job log files. It must
         have access to the Slurm/PBS queue and node status commands (``squeue``, ``sinfo``,
         ``pbsnodes``, ``qstat`` ) to discover partitions and to display cluster usage.

      -  ``group_name``: The group that the Launcher will belong to. It should be a group that is not
            shared with other non-privileged users.

      -  ``singularity_image_root``: The shared directory where Singularity images should be
         located. This directory must be visible to the launcher and from the compute nodes. See
         :ref:`slurm-image-config` for more details.

      -  ``job_storage_root``: The shared directory where job-related files will be stored. It is
         where the needed Determined executables are copied to when the experiment is run, as well
         as where the Slurm/PBS scripts and log files are created. This directory must be writable
         by the launcher and the compute nodes.

      -  ``path``: The ``PATH`` for the launcher service so that it is able to find the Slurm, PBS,
         Singularity, Nvidia binaries, etc., in case they are not in a standard location on the
         compute node. For example, ``PATH=/opt/singularity/3.8.5/bin:${PATH}``.

      -  ``ld_library_path``: The ``LD_LIBRARY_PATH`` for the launcher service so that it is able to
         find the Slurm, PBS, Singularity, Nvidia libraries, etc., in case they are not in a
         standard location on the compute node. For example,
         ``LD_LIBRARY_PATH=/cm/shared/apps/slurm/21.08.6/lib:/cm/shared/apps/slurm/21.08.6/lib/slurm:${LD_LIBRARY_PATH}``.

      -  ``launcher_jvm_args``: Provides an override of the default HPC launcher JVM heap
         configuration.

      -  ``tres_supported``: Indicates if ``SelectType=select/cons_tres`` is set in the Slurm
         configuration. Affects how Determined requests GPUs from Slurm. The default is true.

      -  ``gres_supported``: Indicates if GPU resources are properly configured in the HPC workload
         manager.

         For PBS, the ``ngpus`` option can be used to identify the number of GPUs available on a
         node.

         For Slurm, ``GresTypes=gpu`` is set in the Slurm configuration, and nodes with GPUs have
         properly configured GRES to indicate the presence of any GPUs. The default is true. When
         false, Determined will request ``slots_per_trial`` nodes and utilize only GPU 0 on each
         node. It is the user's responsibility to ensure that GPUs will be available on nodes
         selected for the job using other configurations, such as targeting a specific resource pool
         with only GPU nodes or specifying a Slurm constraint in the experiment configuration.

      -  ``partition_overrides``: A map of partition/queue names to partition-level overrides. For
         each configuration, if it is set for a given partition, it overrides the setting at the
         root level and applies to the resource pool resulting from this partition. Partition names
         are treated as case-insensitive.

         -  ``description`` Description of the resource pool
         -  ``rendezvous_network_interface`` Interface used to bootstrap communication between
            distributed jobs
         -  ``proxy_network_interface`` Interface used to proxy the master for services running on
            compute nodes
         -  ``slot_type`` Resource type used for tasks
         -  ``task_container_defaults`` (See :ref:`top-level setting
            <master-task-container-defaults>`)

         Each ``partition_overrides`` entry may specify a ``task_container_defaults`` that applies
         additional defaults on top of the :ref:`top-level task_container_defaults
         <master-task-container-defaults>` for all tasks launched on that partition. When applying
         the defaults, individual fields override prior values, and list fields are appended. If the
         partition is referenced in a custom HPC resource pool, an additional
         ``task_container_defaults`` may be applied by the resource pool.

         .. code::

            partition_overrides:
               mlde_cuda:
                  description: Partition for CUDA jobs (tesla cards only)
                  slot_type: cuda
                  task_container_defaults:
                     dtrain_network_interface: hsn0,hsn1,hsn2,hsn3
                     slurm:
                        sbatch_args:
                           - --cpus-per-gpu=16
                           - --mem-per-gpu=65536
                        gpu_type: tesla
               mlde_cpu:
                  description: Generic CPU job partition (limited to node001)
                  slot_type: cpu
                  task_container_defaults:
                     slurm:
                        sbatch_args:
                              --nodelist=node001

      -  ``default_aux_resource_pool``: The default resource pool to use for tasks that do not need
         dedicated compute resources, auxiliary, or systems tasks. Defaults to the Slurm/PBS default
         partition if no resource pool is specified.

      -  ``default_compute_resource_pool``: The default resource pool to use for tasks that require
         compute resources, e.g. GPUs or dedicated CPUs. Defaults to the Slurm/PBS default partition
         if it has GPU resources and if no resource pool is specified.

      -  ``job_project_source``: Configures labelling of jobs on the HPC cluster (via Slurm
         ``--wckey`` or PBS ``-P``). Allowed values are:

         -  ``project``: Use the project name of the experiment (this is the default, if no project
            nothing is passed to workload manager).

         -  ``workspace``: Use the workspace name of the project (if no workspace, nothing is passed
            to workload manager).

         -  ``label`` [:``prefix``]: Use the value from the experiment configuration tags list (if
            no matching tags, nothing is passed to workload manager). If a tag begins with the
            specified ``prefix``, remove the prefix and use the remainder as the value for the
            WCKey/Project. If multiple tag values begin with ``prefix``, the remainders are
            concatenated with a comma (,) separator on Slurm or underscore (_) with PBS. If a
            ``prefix`` is not specified or empty, all tags will be matched (and therefore
            concatenated). Workload managers do not generally support multiple WCKey/Project values
            so it is recommended that ``prefix`` is configured to match a single label to enable use
            of the workload manager reporting tools that summarize usage by each WCKey/Project
            value.

.. _cluster-resource-pools:

-  ``resource_pools``: A list of resource pools. A resource pool is a collection of identical
   computational resources. Users can specify which resource pool a job should be assigned to when
   the job is submitted. Refer to the documentation on :ref:`resource-pools` for more information.
   Defaults to a resource pool with a name ``default``.

   -  ``pool_name``: The name of the resource pool.

   -  ``description``: The description of the resource pool.

   -  ``max_aux_containers_per_agent``: The maximum number of auxiliary or system containers that
      can be scheduled on each agent in this pool.

   -  ``agent_reconnect_wait``: Maximum time the master should wait for a disconnected agent before
      considering it dead.

   -  ``agent_reattach_enabled`` (experimental): Whether master & agent try to recover running
      containers after a restart. On master or agent process restart, the agent must reconnect
      within ``agent_reconnect_wait`` period.

   -  ``task_container_defaults``: Each resource pool may specify a ``task_container_defaults`` that
      applies additional defaults on top of the :ref:`top-level setting
      <master-task-container-defaults>` (and ``partition_overrides`` for Slurm/PBS) for all tasks
      launched in that resource pool. When applying the defaults, individual fields override prior
      values, and list fields are appended.

   -  ``kubernetes_namespace``: When the Kubernetes resource manager is in use, this specifies a
      `namespace <https://kubernetes.io/docs/concepts/overview/working-with-objects/namespaces/>`__
      that tasks in this resource pool will be launched into.

   -  ``scheduler``: Specifies how Determined schedules tasks to agents. The scheduler configuration
      on each resource pool will override the global one. For more on scheduling behavior in
      Determined, see :ref:`scheduling`.

      -  ``type``: The scheduling policy to use when allocating resources between different tasks
         (experiments, notebooks, etc.). Defaults to ``fair_share``.

         -  ``fair_share``: Tasks receive a proportional amount of the available resources depending
            on the resource they require and their weight.

         -  ``round_robin``: Tasks are scheduled in the order which they arrive at the cluster.

         -  ``priority``: Tasks are scheduled based on their priority, which can range from the
            values 1 to 99 inclusive. Lower priority numbers indicate higher priority tasks. A lower
            priority task will never be scheduled while a higher priority task is pending. Zero-slot
            tasks (e.g., CPU-only notebooks, tensorboards) are prioritized separately from tasks
            requiring slots (e.g., experiments running on GPUs). Task priority can be assigned using
            the ``resources.priority`` field. If a task does not specify a priority it is assigned
            the ``default_priority``.

            -  ``preemption``: Specifies whether lower priority tasks should be preempted to
               schedule higher priority tasks. Tasks are preempted in order of lowest priority
               first.

            -  ``default_priority``: The priority that is assigned to tasks that do not specify a
               priority. Can be configured to 1 to 99 inclusively. Defaults to ``42``.

      -  ``fitting_policy``: The scheduling policy to use when assigning tasks to agents in the
         cluster. Defaults to ``best``.

         -  ``best``: The best-fit policy ensures that tasks will be preferentially "packed"
            together on the smallest number of agents.
         -  ``worst``: The worst-fit policy ensures that tasks will be placed on under-utilized
            agents.

   -  ``provider``: Specifies the configuration of dynamic agents.

      -  ``master_url``: The full URL of the master. A valid URL is in the format of
         ``scheme://host:port``. The scheme must be either ``http`` or ``https``. If the master is
         deployed on EC2, rather than hardcoding the IP address, we advise you use one of the
         following to set the host as an alias: ``local-ipv4``, ``public-ipv4``, ``local-hostname``,
         or ``public-hostname``. If the master is deployed on GCP, rather than hardcoding the IP
         address, we advise you use one of the following to set the host as an alias:
         ``internal-ip`` or ``external-ip``. Which one you should select is based on your network
         configuration. On master startup, we will replace the above alias host with its real value.
         Defaults to ``http`` as scheme, local IP address as host, and ``8080`` as port.

      -  ``master_cert_name``: A hostname for which the master's TLS certificate is valid, if the
         host specified by the ``master_url`` option is an IP address or is not contained in the
         certificate. See :ref:`tls` for more information.

      -  ``startup_script``: One or more shell commands that will be run during agent instance start
         up. These commands are executed as root as soon as the agent cloud instance has started and
         before the Determined agent container on the instance is launched. For example, this
         feature can be used to mount a distributed file system or make changes to the agent
         instance's configuration. The default value is the empty string. It may be helpful to use
         the YAML ``|`` syntax to specify a multi-line string. For example,

         .. code::

            startup_script: |
                            mkdir -p /mnt/disks/second
                            mount /dev/sdb1 /mnt/disks/second

      -  ``container_startup_script``: One or more shell commands that will be run when the
         Determined agent container is started. These commands are executed inside the agent
         container but before the Determined agent itself is launched. For example, this feature can
         be used to configure Docker so that the agent can pull task images from GCR securely (see
         :ref:`this example <gcp-pull-gcr>` for more details). The default value is the empty
         string.

      -  ``agent_docker_image``: The Docker image to use for the Determined agents. A valid form is
         ``<repository>:<tag>``. Defaults to ``determinedai/determined-agent:<master version>``.

      -  ``agent_docker_network``: The Docker network to use for the Determined agent and task
         containers. If this is set to ``host``, `Docker host-mode networking
         <https://docs.docker.com/network/host/>`__ will be used instead. The default value is
         ``determined``.

      -  ``agent_docker_runtime``: The Docker runtime to use for the Determined agent and task
         containers. Defaults to ``runc``.

      -  ``max_idle_agent_period``: How long to wait before terminating idle dynamic agents. This
         string is a sequence of decimal numbers, each with optional fraction and a unit suffix,
         such as "30s", "1h", or "1m30s". Valid time units are "s", "m", "h". The default value is
         ``20m``.

      -  ``max_agent_starting_period``: How long to wait for agents to start up before retrying.
         This string is a sequence of decimal numbers, each with optional fraction and a unit
         suffix, such as "30s", "1h", or "1m30s". Valid time units are "s", "m", "h". The default
         value is ``20m``.

      -  ``min_instances``: Min number of Determined agent instances. Defaults to ``0``.

      -  ``max_instances``: Max number of Determined agent instances. Defaults to ``5``.

      -  ``type: aws``: Specifies running dynamic agents on AWS. (*Required*)

         -  ``region``: The region of the AWS resources used by Determined. We advise setting this
            region to be the same region as the Determined master for better network performance.
            Defaults to the same region as the master.

         -  ``root_volume_size``: Size of the root volume of the Determined agent in GB. We
            recommend at least 100GB. Defaults to ``200``.

         -  ``image_id``: The AMI ID of the Determined agent. Defaults to the latest GCP agent
            image. (*Optional*)

         -  ``tag_key``: Key for tagging the Determined agent instances. Defaults to ``managed-by``.

         -  ``tag_value``: Value for tagging the Determined agent instances. Defaults to the master
            instance ID if the master is on EC2, otherwise ``determined-ai-determined``.

         -  ``custom_tags``: List of arbitrary user-defined tags that are added to the Determined
            agent instances and do not affect how Determined works. Each tag must specify ``key``
            and ``value`` fields. Defaults to the empty list.

            -  ``key``: Key of custom tag.
            -  ``value``: value of custom tag.

         -  ``instance_name``: Name to set for the Determined agent instances. Defaults to
            ``determined-ai-agent``.

         -  ``ssh_key_name``: The name of the SSH key registered with AWS for SSH key access to the
            agent instances. (*Required*)

         -  ``iam_instance_profile_arn``: The Amazon Resource Name (ARN) of the IAM instance profile
            to attach to the agent instances.

         -  ``network_interface``: Network interface to set for the Determined agent instances.

            -  ``public_ip``: Whether to use public IP addresses for the Determined agents. See
               :ref:`aws-network-requirements` for instructions on whether a public IP should be
               used. Defaults to ``true``.

            -  ``security_group_id``: The ID of the security group to run the Determined agents as.
               This should be the security group you identified or created in
               :ref:`aws-network-requirements`. Defaults to the default security group of the
               specified VPC.

            -  ``subnet_id``: The ID of the subnet to run the Determined agents in. Defaults to the
               default subnet of the default VPC.

         -  ``instance_type``: AWS instance type to use for dynamic agents. If ``instance_slots`` is
            not specified, for GPU instances this must be one of the following: ``g4dn.xlarge``,
            ``g4dn.2xlarge``, ``g4dn.4xlarge``, ``g4dn.8xlarge``, ``g4dn.16xlarge``,
            ``g4dn.12xlarge``, ``g4dn.metal``, ``g5.xlarge``, ``g5.2xlarge``, ``g5.4xlarge``,
            ``g5.8xlarge``, ``g5.12xlarge``, ``g5.16xlarge``, ``g5.24xlarge``, ``g5.48large``,
            ``p3.2xlarge``, ``p3.8xlarge``, ``p3.16xlarge``, ``p3dn.24xlarge``, or ``p4d.24xlarge``.
            For CPU instances, most general purpose instance types are allowed (``t2``, ``t3``,
            ``c4``, ``c5``, ``m4``, ``m5`` and variants). Defaults to ``g4dn.metal``.

         -  ``instance_slots``: The optional number of GPUs for the AWS instance type. This is used
            in conjunction with the ``instance_type`` in order to specify types which are not listed
            in the ``instance_type`` list above. Note that some GPUs may not be supported.
            **WARNING**: *be sure to specify the correct number of GPUs to ensure that provisioner
            launches the correct number of instances.*

         -  ``cpu_slots_allowed``: Whether to allow slots on the CPU instance types. When ``true``,
            and if the instance type doesn't have any GPUs, each instance will provide a single
            CPU-based compute slot; if it has any GPUs, they'll be used for compute slots instead.
            Defaults to ``false``.

         -  ``spot``: Whether to use spot instances. Defaults to ``false``. See :ref:`aws-spot` for
            more details.

         -  ``spot_max_price``: Optional field indicating the maximum price per hour that you are
            willing to pay for a spot instance. The market price for a spot instance varies based on
            supply and demand. If the market price exceeds the ``spot_max_price``, Determined will
            not launch instances. This field must be a string and must not include a currency sign.
            For example, $2.50 should be represented as ``"2.50"``. Defaults to the on-demand price
            for the given instance type.

      -  ``type: gcp``: Specifies running dynamic agents on GCP. (*Required*)

         -  ``base_config``: Instance resource base configuration that will be merged with the
            fields below to construct GCP inserting instance request. See `REST Resource: instances
            <https://cloud.google.com/compute/docs/reference/rest/v1/instances/insert>`__ for
            details.

         -  ``project``: The project ID of the GCP resources used by Determined. Defaults to the
            project of the master.

         -  ``zone``: The zone of the GCP resources used by Determined. Defaults to the zone of the
            master.

         -  ``boot_disk_size``: Size of the root volume of the Determined agent in GB. We recommend
            at least 100GB. Defaults to ``200``.

         -  ``boot_disk_source_image``: The boot disk source image of the Determined agent that was
            shared with you. To use a specific version of the Determined agent image from a specific
            project, it should be set in the format:
            ``projects/<project-id>/global/images/<image-id>``. Defaults to the latest GCP agent
            image. (*Optional*)

         -  ``label_key``: Key for labeling the Determined agent instances. Defaults to
            ``managed-by``.

         -  ``label_value``: Value for labeling the Determined agent instances. Defaults to the
            master instance name if the master is on GCP, otherwise ``determined-ai-determined``.

         -  ``name_prefix``: Name prefix to set for the Determined agent instances. The names of the
            Determined agent instances are a concatenation of the name prefix and a pet name.
            Defaults to the master instance name if the master is on GCP otherwise
            ``determined-ai-determined``.

         -  ``network_interface``: Network configuration for the Determined agent instances. See the
            :ref:`gcp-api-access` section for the suggested configuration. (*Required*)

            -  ``network``: Network resource for the Determined agent instances. The network
               configuration should specify the project ID of the network. It should be set in the
               format: ``projects/<project>/global/networks/<network>``. (*Required*)

            -  ``subnetwork``: Subnetwork resource for the Determined agent instances. The subnet
               configuration should specify the project ID and the region of the subnetwork. It
               should be set in the format:
               ``projects/<project>/regions/<region>/subnetworks/<subnetwork>``. (*Required*)

            -  ``external_ip``: Whether to use external IP addresses for the Determined agent
               instances. See :ref:`gcp-network-requirements` for instructions on whether an
               external IP should be set. Defaults to ``false``.

         -  ``network_tags``: An array of network tags to set firewalls for the Determined agent
            instances. This is the one you identified or created in :ref:`firewall-rules`. Defaults
            to be an empty array.

         -  ``service_account``: Service account for the Determined agent instances. See the
            :ref:`gcp-api-access` section for suggested configuration.

            -  ``email``: Email of the service account for the Determined agent instances. Defaults
               to the empty string.

            -  ``scopes``: List of scopes authorized for the Determined agent instances. As
               suggested in :ref:`gcp-api-access`, we recommend you set the scopes to
               ``["https://www.googleapis.com/auth/cloud-platform"]``. Defaults to
               ``["https://www.googleapis.com/auth/cloud-platform"]``.

         -  ``instance_type``: Type of instance for the Determined agents.

            -  ``machine_type``: Type of machine for the Determined agents. Defaults to
               ``n1-standard-32``.
            -  ``gpu_type``: Type of GPU for the Determined agents. Set it to be an empty string to
               not use any GPUs. Defaults to ``nvidia-tesla-t4``.
            -  ``gpu_num``: Number of GPUs for the Determined agents. Defaults to 4.
            -  ``preemptible``: Whether to use preemptible dynamic agent instances. Defaults to
               ``false``.

         -  ``cpu_slots_allowed``: Whether to allow slots on the CPU instance types. When ``true``,
            and if the instance type doesn't have any GPUs, each instance will provide a single
            CPU-based compute slot; if it has any GPUs, they'll be used for compute slots instead.
            Defaults to ``false``.

         -  ``operation_timeout_period``: The timeout period for tracking a GCP operation. This
            string is a sequence of decimal numbers, each with optional fraction and a unit suffix,
            such as "30s", "1h", or "1m30s". Valid time units are "s", "m", "h". The default value
            is ``5m``.

      -  ``type: hpc``: Specifies a custom resource pool that submits work to an underlying
         Slurm/PBS partition on an HPC cluster. (*Required*)

         One resource pool is automatically created for each Slurm partition or PBS queue on an HPC
         cluster. This provider enables the creation of additional resource pools with different
         submission options to those partitions/queues.

         -  ``partition``: The target HPC partition where jobs will be launched when using this
            resource pool. Add ``task_container_defaults`` to provide a resource pool with
            additional default options. The ``task_container_defaults`` from the resource pool are
            applied after any ``task_container_defaults`` from ``partition_overrides``. When
            applying the defaults, individual fields override prior values, and list fields are
            appended. This can be used to create a resource pool with homogeneous resources when the
            underlying partition or queue does not. Consider the following:

         .. code::

            resource_pools:
              - pool_name: defq_GPU_tesla
                description: Lands jobs on defq_GPU with tesla GPU selected, XL675d systems
                task_container_defaults:
                  slurm:
                    gpu_type: tesla
                    sbatch_options:
                      - -CXL675d
                provider:
                  type: hpc
                  partition: defq_GPU

         In this example, jobs submitted to the resource pool named ``defq_GPU_tesla`` will be
         executed in the HPC partition named ``defq_GPU`` with the ``gpu_type`` property set, and
         Slurm constraint associated with the feature ``XL675d`` used to identify the model type of
         the compute node.

-  ``checkpoint_storage``: Specifies where model checkpoints will be stored. This can be overridden
   on a per-experiment basis in the :ref:`experiment-configuration`. A checkpoint contains the
   architecture and weights of the model being trained. Determined currently supports several kinds
   of checkpoint storage, ``gcs``, ``hdfs``, ``s3``, ``azure``, and ``shared_fs``, identified by the
   ``type`` subfield.

   -  ``type: gcs``: Checkpoints are stored on Google Cloud Storage (GCS). Authentication is done
      using GCP's "`Application Default Credentials
      <https://googleapis.dev/python/google-api-core/latest/auth.html>`__" approach. When using
      Determined inside Google Compute Engine (GCE), the simplest approach is to ensure that the VMs
      used by Determined are running in a service account that has the "Storage Object Admin" role
      on the GCS bucket being used for checkpoints. As an alternative (or when running outside of
      GCE), you can add the appropriate `service account credentials
      <https://cloud.google.com/docs/authentication/production#obtaining_and_providing_service_account_credentials_manually>`__
      to your container (e.g., via a bind-mount), and then set the
      ``GOOGLE_APPLICATION_CREDENTIALS`` environment variable to the container path where the
      credentials are located. See :ref:`environment-variables` for more information on how to set
      environment variables in trial environments.

      -  ``bucket``: The GCS bucket name to use.
      -  ``prefix``: The optional path prefix to use. Must not contain ``..``. Note: Prefix is
         normalized, e.g., ``/pre/.//fix`` -> ``/pre/fix``

   -  ``type: hdfs``: Checkpoints are stored in HDFS using the `WebHDFS
      <http://hadoop.apache.org/docs/current/hadoop-project-dist/hadoop-hdfs/WebHDFS.html>`__ API
      for reading and writing checkpoint resources.

      -  ``hdfs_url``: Hostname or IP address of HDFS namenode, prefixed with protocol, followed by
         WebHDFS port on namenode. Multiple namenodes are allowed as a semicolon-separated list
         (e.g., ``"http://namenode1:50070;http://namenode2:50070"``).

      -  ``hdfs_path``: The prefix path where all checkpoints will be written to and read from. The
         resources of each checkpoint will be saved in a subdirectory of ``hdfs_path``, where the
         subdirectory name is the checkpoint's UUID.

      -  ``user``: An optional string value that indicates the user to use for all read and write
         requests. If left unspecified, the default user of the trial runner container will be used.

   -  ``type: s3``: Checkpoints are stored in Amazon S3.

      -  ``bucket``: The S3 bucket name to use.
      -  ``access_key``: The AWS access key to use.
      -  ``secret_key``: The AWS secret key to use.
      -  ``prefix``: The optional path prefix to use. Must not contain ``..``. Note: Prefix is
         normalized, e.g., ``/pre/.//fix`` -> ``/pre/fix``
      -  ``endpoint_url``: The optional endpoint to use for S3 clones, e.g.,
         ``http://127.0.0.1:8080/``.

   -  ``type: azure``: Checkpoints are stored in Microsoft's Azure Blob Storage. Authentication is
      performed by providing either a connection string, or an account URL and an optional
      credential.

      -  ``container``: The Azure Blob Storage container name to use.
      -  ``connection_string``: The connection string for the service account to use.
      -  ``account_url``: The account URL for the service account to use.
      -  ``credential``: The optional credential to use in conjunction with the account URL.

      Please only specify either ``connection_string`` or the ``account_url`` and ``credential``
      pair.

   -  ``type: shared_fs``: Checkpoints are written to a directory on the agent's file system. The
      assumption is that the system administrator has arranged for the same directory to be mounted
      at every agent host, and for the content of this directory to be the same on all agent hosts
      (e.g., by using a distributed or network file system such as GlusterFS or NFS).

      -  ``host_path``: The file system path on each agent to use. This directory will be mounted to
         ``/determined_shared_fs`` inside the trial container.

      -  ``storage_path``: The optional path where checkpoints will be written to and read from.
         Must be a subdirectory of the ``host_path`` or an absolute path containing the
         ``host_path``. If unset, checkpoints are written to and read from the ``host_path``.

      -  ``propagation``: (Advanced users only) Optional `propagation behavior
         <https://docs.docker.com/storage/bind-mounts/#configure-bind-propagation>`__ for replicas
         of the bind-mount. Defaults to ``rprivate``.

   -  When an experiment finishes, the system will optionally delete some checkpoints to reclaim
      space. The ``save_experiment_best``, ``save_trial_best`` and ``save_trial_latest`` parameters
      specify which checkpoints to save. See :ref:`checkpoint-garbage-collection` for more details.

-  ``db``: Specifies the configuration of the database.

   -  ``user``: The database user to use when logging in the database. (*Required*)

   -  ``password``: The password to use when logging in the database. (*Required*)

   -  ``host``: The database host to use. (*Required*)

   -  ``port``: The database port to use. (*Required*)

   -  ``name``: The database name to use. (*Required*)

   -  ``ssl_mode``: The SSL mode to use. See the `PostgreSQL documentation
      <https://www.postgresql.org/docs/current/libpq-ssl.html#LIBPQ-SSL-SSLMODE-STATEMENTS>`__ for
      the list of possible values and their meanings. Defaults to ``disable``. In order to ensure
      that SSL is used, this should be set to ``require``, ``verify-ca``, or ``verify-full``.

   -  ``ssl_root_cert``: The location of the root certificate file to use for verifying the server's
      certificate. See the `PostgreSQL documentation
      <https://www.postgresql.org/docs/current/libpq-ssl.html#LIBQ-SSL-CERTIFICATES>`__ for more
      information about certificate verification. Defaults to ``~/.postgresql/root.crt``.

-  ``security``: Specifies security-related configuration settings.

   -  ``tls``: Specifies configuration settings for :ref:`TLS <tls>`. TLS is enabled if certificate
      and key files are both specified.

      -  ``cert``: Certificate file to use for serving TLS.
      -  ``key``: Key file to use for serving TLS.

   -  ``ssh``: Specifies configuration settings for SSH.

      -  ``rsa_key_size``: Number of bits to use when generating RSA keys for SSH for tasks. Maximum
         size is 16384.

   -  ``authz``: Authorization settings.

         -  ``type``: Authorization system to use. Defaults to ``basic``. See :ref:`RBAC docs
            <rbac>` for further info.

         -  ``rbac_ui_enabled``: Whether to enable RBAC in WebUI and CLI. When ``type`` is ``rbac``,
            defaults ``true``, otherwise ``false``.

         -  ``workspace_creator_assign_role``: Assign a role to the user on workspace creation.

               -  ``enabled``: Whether this feature is enabled. Defaults to ``true``.
               -  ``role_id``: Integer identifier of a role to be assigned. Defaults to ``2``, which
                  is the role id of ``WorkspaceAdmin`` role.

-  ``webhooks``: Specifies configuration settings related to webhooks.

   -  ``signing_key``: The key used to sign outgoing webhooks.
   -  ``base_url``: The URL users use to access Determined, for generating hyperlinks.

-  ``telemetry``: Specifies configuration settings related to telemetry collection and tracing.

   -  ``enabled``: Whether to collect and report anonymous information about the usage of this
      Determined cluster. See :ref:`telemetry` for details on what kinds of information are
      reported. Defaults to ``true``.

   -  ``otel-enabled``: Whether OpenTelemetry is enabled. Defaults to ``false``.

   -  ``otel-endpoint``: OpenTelemetry endpoint to use. Defaults to ``localhost:4317``.

-  ``observability``: Specifies whether Determined enables Prometheus monitoring routes. See
   :ref:`Prometheus <prometheus>` for details.

   -  ``enable_prometheus``: Whether Prometheus is enabled. Defaults to ``false``.

-  ``logging``: Specifies configuration settings for the logging backend for trial logs.

   -  ``type: default``: Trial logs are shipped to the master and stored in Postgres. If nothing is
      set, this is the default.

   -  ``type: elastic``: Trial logs are shipped to the Elasticsearch cluster described by the
      configuration settings in the section. See :ref:`the topic guide
      <elasticsearch-logging-backend>` for a more detailed explanation of how and when to use
      Elasticsearch.

      -  ``host``: Hostname or IP address for the cluster.

      -  ``port``: Port for the cluster.

      -  ``security``: Security-related configuration settings.

         -  ``username``: Username to use when accessing the cluster.

         -  ``password``: Password to use when accessing the cluster.

         -  ``tls``: TLS-related configuration settings.

            -  ``enabled``: Enable TLS.

            -  ``skip_verify``: Skip server certificate verification.

            -  ``certificate``: Path to a file containing the cluster's TLS certificate. Only needed
               if the certificate is not signed by a well-known CA; cannot be specified if
               ``skip_verify`` is enabled.

   -  ``additional_fluent_outputs``: An optional configuration string containing additional Fluent
      Bit outputs for advanced users to specify logging integrations. See the `Fluent Bit
      documentation <https://docs.fluentbit.io/manual/pipeline/outputs>`__ for the format and
      supported logging outputs.

-  ``scim``: (EE-only) Specifies whether the SCIM service is enabled and the credentials for clients
   to use it.

   -  ``enabled``: Whether to enable SCIM. Defaults to ``false``.

   -  ``auth``: The configuration for authenticating SCIM requests.

      -  ``type``: The authentication type to use. Either ``"basic"`` (for HTTP basic
         authentication) or ``"oauth"`` (for :ref:`OAuth 2.0 <oauth>`).
      -  ``username``: The username for HTTP basic authentication (only allowed with ``type:
         basic``).
      -  ``password``: The password for HTTP basic authentication (only allowed with ``type:
         basic``).

-  ``saml``: (EE-only) Specifies whether SAML SSO is enabled and the configuration to use it.

   -  ``enabled``: Whether to enable SAML SSO. Defaults to ``false``.
   -  ``provider``: The name of the IdP. Currently (officially) supported: "okta".
   -  ``idp_recipient_url``: The URL your IdP will send SAML assertions to.
   -  ``idp_sso_url``: An IdP-provided URL to redirect SAML requests to.
   -  ``idp_sso_descriptor_url``: An IdP-provided URL, also known as IdP issuer. It is an identifier
      for the IdP that issues the SAML requests and responses.
   -  ``idp_cert_path``: The path to the IdP's certificate, used to validate assertions.
