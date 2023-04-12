.. _command-notebook-configuration:

.. _job-configuration-reference:

#############################
 Job Configuration Reference
#############################

The behavior of interactive jobs, such as :ref:`TensorBoards <tensorboards>`, :ref:`notebooks
<notebooks>`, :ref:`commands, and shells <commands-and-shells>`, can be influenced by setting a
variety of configuration variables. These configuration variables are similar but not identical to
the configuration options supported by :ref:`experiments <experiment-config-reference>`.

Configuration settings can be specified by passing a YAML configuration file when launching the
workload via the Determined CLI:

.. code::

   $ det tensorboard start experiment_id --config-file=my_config.yaml
   $ det notebook start --config-file=my_config.yaml
   $ det cmd run --config-file=my_config.yaml ...
   $ det shell start --config-file=my_config.yaml

Configuration variables can also be set directly on the command line when any Determined task,
except a TensorBoard, is launched:

.. code::

   $ det notebook start --config resources.slots=2
   $ det cmd run --config description="determined_command" ...
   $ det shell start --config resources.priority=1

Options set via ``--config`` take precedence over values specified in the configuration file.
Configuration settings are compatible with any Determined task unless otherwise specified.

The following configuration settings are supported:

-  ``description``: A human-readable description of the task. This does not need to be unique. The
   default description consists of a timestamp and the entrypoint of the command.

-  ``environment``: Specifies the environment of the container that is used to execute the task.

   -  ``image``: The Docker image to use when executing the workload. This image must be accessible
      via ``docker pull`` to every Determined agent machine in the cluster. Users can configure
      different container images for NVIDIA GPU tasks using ``cuda`` key (``gpu`` prior to 0.17.6),
      CPU tasks using ``cpu`` key, and ROCm (AMD GPU) tasks using ``rocm`` key. Default values:

      -  ``determinedai/environments-dev:cuda-11.3-pytorch-1.12-tf-2.8-gpu-0.21.2`` for NVIDIA GPUs.
      -  ``determinedai/environments-dev:rocm-5.0-pytorch-1.10-tf-2.7-rocm-0.21.2`` for ROCm.
      -  ``determinedai/environments-dev:py-3.8-pytorch-1.12-tf-2.8-cpu-0.21.2`` for CPUs.

   -  ``force_pull_image``: Forcibly pull the image from the Docker registry and bypass the Docker
      cache. Defaults to ``false``.

   -  ``environment_variables``: A list of environment variables that will be set in every trial
      container. Each element of the list should be a string of the form ``NAME=VALUE``. See
      :ref:`environment-variables` for more details. Users can customize environment variables for
      GPU, CPU, and ROCm tasks differently by specifying a dict with ``cuda`` (``gpu`` prior to
      0.17.6), ``cpu``, and ``rocm`` keys.

   -  ``pod_spec``: Only applicable when running Determined on Kubernetes. Applies a pod spec to the
      pods that are launched by Determined for this task. See :ref:`custom-pod-specs` for details.

   -  ``registry_auth``: Specifies the `Docker registry credentials
      <https://docs.docker.com/engine/api/v1.30/#operation/SystemAuth>`__ to use when pulling a
      Docker image, if needed.

      -  ``username`` (required)
      -  ``password`` (required)
      -  ``server`` (optional)
      -  ``email`` (optional)

   -  ``add_capabilities``: A list of Linux capabilities to grant to task containers. Each entry in
      the list is equivalent to a ``--cap-add CAP`` command-line argument to ``docker run``.
      ``add_capabilities`` is honored by resource managers of type ``agent`` but is ignored by
      resource managers of type ``kubernetes``. See :ref:`master configuration
      <master-config-reference>` for details about resource managers.

   -  ``drop_capabilities``: Just like ``add_capabilities`` but corresponding to the ``--cap-drop``
      argument of ``docker run`` rather than ``--cap-add``.

   -  ``proxy_ports``: Expose configured network ports on the chief task container. See
      :ref:`proxy-ports` for details.

-  ``resources``: The resources Determined allows a task to use.

   -  ``slots``: Specifies the number of slots to use for the task. The default value is ``1``. The
      maximum value is the number of slots on the agent in the cluster with the most slots. For
      example, Determined will be unable to schedule a task that requests 4 slots if the Determined
      cluster is composed of agents with 2 slots each. The number of slots for TensorBoard is fixed
      at ``0`` and may not be changed.

   -  ``shm_size``: The size of ``/dev/shm`` for task containers. The value can be a number in bytes
      or a number with a suffix (e.g., ``128M`` for 128MiB or ``1.5G`` for 1.5GiB). Defaults to
      ``4294967296`` (4GiB). If set, this value overrides the value specified in the :ref:`master
      configuration <master-config-reference>`.

   -  ``priority``: The priority assigned to this task. Tasks with smaller priority values are
      scheduled before tasks with higher priority values. Only applicable when using the
      ``priority`` scheduler. Refer to :ref:`scheduling` for more information.

   -  ``resource_pool``: The resource pool where this task will be scheduled. If no resource pool is
      specified, CPU-only tasks will be scheduled in the default CPU pool, while GPU-using tasks
      will be scheduled in the default GPU tool. Refer to :ref:`resource-pools` for more
      information.

   -  ``devices``: A list of device strings to pass to the Docker daemon. Each entry in the list is
      equivalent to a ``--device DEVICE`` command-line argument to ``docker run``. ``devices`` is
      honored by resource managers of type ``agent`` but is ignored by resource managers of type
      ``kubernetes``. See :ref:`master configuration <master-config-reference>` for details about
      resource managers.

   -  ``agent_label``: This field has been deprecated and will be ignored. Use ``resource_pool``
      instead.

-  ``bind_mounts``: Specifies a collection of directories that are bind-mounted into the Docker
   containers for execution. This can be used to allow commands to access additional data that is
   not contained in the command context. This field should consist of an array of entries. Note that
   users should ensure that the specified host paths are accessible on all agent hosts (e.g., by
   configuring a network file system appropriately). Defaults to an empty list.

   -  ``host_path``: (required) The file system path on each agent to use. Must be an absolute
      filepath.

   -  ``container_path``: (required) The file system path in the container to use. May be a relative
      filepath, in which case it will be mounted relative to the working directory inside the
      container. It is not allowed to mount directly into the working directory (``container_path ==
      "."``) to reduce the risk of cluttering the host filesystem.

   -  ``read_only``: Whether the bind-mount should be a read-only mount. Defaults to ``false``.

   -  ``propagation``: (Advanced users only) Optional `propagation behavior
      <https://docs.docker.com/storage/bind-mounts/#configure-bind-propagation>`__ for replicas of
      the bind-mount. Defaults to ``rprivate``.

-  ``work_dir``: Working directory. This can include ``$AGENT_USER`` or ``$DET_USER``, which will be
   replaced with the actual agent user id or determined user id. This cannot be set if submitting a
   context directory. Defaults to null.

-  ``tensorboard_args``: Lists optional arguments for launching TensorBoard. Each element of the
   list should be a string of the form ``NAME=VALUE``.

-  ``idle_timeout``: Specifies the duration before idle instances are automatically terminated. This
   string is a sequence of decimal numbers, each with optional fraction and a unit suffix, such as
   "30s", "1h", or "1m30s". Valid time units are "s", "m", "h". The default value is ``20m``. This
   is only used by TensorBoard and notebook instances. A TensorBoard instance is considered to be
   idle if it does not receive any HTTP traffic. A notebook instance is considered to be idle if it
   is not receiving any HTTP traffic and it is not otherwise active (as defined by the
   ``notebook_idle_type`` option). The default timeout for TensorBoard is ``5m`` (5 minutes).

-  ``notebook_idle_type``: Specifies how to decide whether a notebook is idle or active. Valid
   values are:

   -  ``kernels_or_terminals`` (default): The notebook is considered active if any kernels or
      terminals are running.

   -  ``kernel_connections``: The notebook is considered active if there are any open connections
      from any web connections to any kernels. (JupyterLab does not report connections to terminals,
      so they cannot be counted.)

   -  ``activity``: The notebook is considered active if any kernel is executing a command or any
      terminal that is currently being viewed in JupyterLab is inputting or outputting any data. (A
      terminal that is running a command but not being viewed or running a command with no output is
      treated as idle, since JupyterLab does not provide activity information for those case.)

-  ``slurm``: Slurm cluster details may optionally be specified in the same fashion as for
   :ref:`experiments <slurm-config>`.

-  ``pbs``: PBS cluster details may optionally be specified in the same fashion as for
   :ref:`experiments <pbs-config>`.
