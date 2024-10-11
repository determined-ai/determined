.. _helm-config-reference:

####################################
 Helm Chart Configuration Reference
####################################

+-----------------------------------------------------------------------------------------+
| Installation Guide                                                                      |
+=========================================================================================+
| :ref:`install-on-kubernetes`                                                            |
+-----------------------------------------------------------------------------------------+

*************************
 ``Chart.yaml`` Settings
*************************

-  ``appVersion``: Configures which version of Determined to install. Users can specify a release
   version (e.g., ``0.13.0``) or specify any commit hash from the `upstream Determined repo
   <https://github.com/determined-ai/determined>`_ (e.g.,
   ``b13461ed06f2fad339e179af8028d4575db71a81``). Users are encouraged to use a released version.

.. note::

   If using a non-release branch of the Determined repository, ``appVersion`` is going to be set to
   ``X.Y.Z.dev0``. This is not an official release version and deploying this version result in a
   ``ImagePullBackOff`` error. Users should remove ``.dev0`` to get the latest released version, or
   they can specify a specific commit hash instead.

**************************
 ``values.yaml`` Settings
**************************

-  ``masterPort``: The port at which the Determined master listens for connections on. (*Required*)

-  ``useNodePortForMaster``: When set to false (default), a LoadBalancer service is deployed to make
   the Determined master reachable from outside the cluster. When set to true, the master will
   instead be exposed behind a NodePort service. When using a NodePort service users will typically
   have to configure an Ingress to make the Determined master reachable from outside the cluster.
   NodePort service is recommended when configuring TLS termination in a load-balancer.

-  ``tlsSecret``: enables TLS encryption for all communication with the Determined master (TLS
   termination is performed in the Determined master). This includes communication between the
   Determined master and the task containers it launches, but does not include communication between
   the task containers (distributed training). The specified Secret of type tls must already exist
   in the same namespace in which Determined is being installed.

-  ``db``: Specifies the configuration of the database.

   -  ``name``: The database name to use. (*Required*)

   -  ``user``: The database user to use when logging in the database. (*Required*)

   -  ``password``: The password to use when logging in the database. (*Required*)

   -  ``port``: The database port to use. (*Required*)

   -  ``hostAddress``: Optional configuration to indicate the address of a user provisioned database
      If configured, the Determined helm chart will not deploy a database.

   -  ``storageSize``: Only used when ``hostAddress`` is left blank. Configures the size of the
      PersistentVolumeClaim for the Determined deployed database.

   -  ``cpuRequest``: The CPU requirements for the Determined database.

   -  ``memRequest``: The memory requirements for the Determined database.

   -  ``cpuLimit``: Optional configuration. The CPU limits for the Determined database.

   -  ``memLimit``: Optional configuration. The memory limits for the Determined database.

   -  ``useNodePortForDB``: Optional configuration that configures whether ClusterIP or NodePort
      service type is used for the Determined database. By default ClusterIP is used.

   -  ``storageClassName``: Optional configuration to indicate StorageClass that should be used by
      the PersistentVolumeClaim for the Determined deployed database. This can be left blank if a
      default storage class is specified in the cluster. If dynamic provisioning of
      PersistentVolumes is disabled, users must manually create a PersistentVolume that will match
      the PersistentVolumeClaim.

   -  ``claimSuffix``: Optional configuration that defines the persistent volume claim name for the
      Determined database. If not undefined, a new PVC is created, with the name
      ``determined-db-pvc-$releaseName``. If specified, the provided value will be used as the
      suffix to ``determined-db-pvc-``.

   -  ``snapshotSuffix``: Optional configuration for naming the volume snapshot, which serves as a
      backup of the Determined database's persistentVolume. If defined, Helm will create a snapshot
      of the database during the next upgrade, creating a volumeSnapshot named
      ``determined-db-snapsnot-$snapshotSuffix``. This **must** **not** match the name of an
      existing persistent volume claim, such as ``determined-db-pvc-$releaseName``, because the
      ``snapshotSuffix`` will be used to create a new persistent volume claim when restoring.

   -  ``restoreSnapshotSuffix``: Optional configuration of the volumeSnapshot to restore from during
      an upgrade. If this is set during an upgrade, a new persistentVolume will be created and
      swapped into Determined's database by creating a new persistent volume claim named
      ``determined-db-pvc-$restoreSnapshotSuffix``, restoring data from
      ``determined-db-snapsnot-$restoreSnapshotSuffix``. This **must not** match the name of an
      existing persistent volume claim (PVC), such as ``$releaseName``.

-  ``checkpointStorage``: Specifies where model checkpoints will be stored. This can be overridden
   on a per-experiment basis in the :ref:`experiment-config-reference`. A checkpoint contains the
   architecture and weights of the model being trained. Determined currently supports several kinds
   of checkpoint storage, ``gcs``, ``s3``, ``azure``, ``shared_fs``, and ``directory``, identified
   by the ``type`` subfield.

   -  ``type: gcs``: Checkpoints are stored on Google Cloud Storage (GCS). Authentication is done
      using GCP's "`Application Default Credentials
      <https://googleapis.dev/python/google-api-core/latest/auth.html>`__" approach. When using
      Determined inside Google Kubernetes Engine (GCE), the simplest approach is to ensure that the
      nodes used by Determined are running in a service account that has the "Storage Object Admin"
      role on the GCS bucket being used for checkpoints. As an alternative (or when running outside
      of GKE), you can add the appropriate `service account credentials
      <https://cloud.google.com/docs/authentication/provide-credentials-adc#attached-sa>`__ to your
      container (e.g., via a bind-mount), and then set the ``GOOGLE_APPLICATION_CREDENTIALS``
      environment variable to the container path where the credentials are located. See
      :ref:`environment-variables` for more information on how to set environment variables in trial
      environments.

      -  ``bucket``: The GCS bucket name to use.
      -  ``prefix``: The optional path prefix to use. Must not contain ``..``. Note: Prefix is
         normalized, e.g., ``/pre/.//fix`` -> ``/pre/fix``

   -  ``type: s3``: Checkpoints are stored in Amazon S3.

      -  ``bucket``: The S3 bucket name to use.
      -  ``accessKey``: The AWS access key to use.
      -  ``secretKey``: The AWS secret key to use.
      -  ``prefix``: The optional path prefix to use. Must not contain ``..``. Note: Prefix is
         normalized, e.g., ``/pre/.//fix`` -> ``/pre/fix``
      -  ``endpointUrl``: The optional endpoint to use for S3 clones, e.g., http://127.0.0.1:8080/.

   -  ``type: azure``: Checkpoints are stored in Microsoft's Azure Blob Storage. Authentication is
      performed by providing either a connection string, or an account URL and an optional
      credential.

      -  ``container``: The Azure Blob Storage container name to use.
      -  ``connection_string``: The connection string for the service account to use.
      -  ``account_url``: The account URL for the service account to use.
      -  ``credential``: The optional credential to use in conjunction with the account URL.

      Please only specify either ``connection_string`` or the ``account_url`` and ``credential``
      pair.

   -  ``type: shared_fs``: Checkpoints are written to a ``hostPath Volume``. Users are strongly
      discouraged from using ``shared_fs`` for storage beyond initial testing as most Kubernetes
      cluster nodes do not have a shared file system.

      -  ``hostPath``: The file system path on each node to use. This directory will be mounted to
         ``/determined_shared_fs`` inside the trial pod.

      -  ``mountToServer``: Allows users to download checkpoints.

         -  ``false``: Default.
         -  ``true``: Mounts the shared directory on the server to allow ``checkpoint.download()``
            to work.

   -  ``type: directory``: Checkpoints are written to a local directory in the task container. Users
      are responsible for mounting a persistent storage at this path, e.g., a shared PVC using
      ``pod_spec`` configuration.

      -  ``containerPath``: The file system path inside the task pods to use. The checkpoints and
         TensorBoard data will be stored under this path.

   -  When an experiment finishes, the system will optionally delete some checkpoints to reclaim
      space. The ``saveExperimentBest``, ``saveTrialBest`` and ``saveTrialLatest`` parameters
      specify which checkpoints to save. See :ref:`checkpoint-garbage-collection` for more details.

-  ``maxSlotsPerPod``: Specifies number of GPUs there are per machine. Determined uses this
   information when scheduling multi-GPU tasks. Each multi-GPU (distributed training) task will be
   scheduled as a set of ``slotsPerTask / maxSlotsPerPod`` separate pods, with each pod assigned up
   to ``maxSlotsPerPod`` GPUs. Distributed tasks with sizes that are not divisible by
   ``maxSlotsPerPod`` are never scheduled. If you have a cluster of different size nodes, set the
   ``maxSlotsPerPod`` to greatest common divisor of all the sizes. For example, if you have some
   nodes with 4 GPUs and other nodes with 8 GPUs, set ``maxSlotsPerPod`` to ``4`` so that all
   distributed experiments will launch with 4 GPUs per pod (with two pods on 8-GPU nodes).
   (*Required*)

-  ``slotType``: Specifies the type of the slot that Determined will deploy. Valid options are
   ``cpu`` for CPU training, ``gpu`` or ``cuda`` for Nvidia GPUs, and ``rocm`` for AMD GPUs.
   Defaults to ``gpu``.

   The ``rocm`` slot type is an experimental feature.

-  ``masterCpuRequest``: The CPU requirements for the Determined master.

-  ``masterMemRequest``: The memory requirements for the Determined master.

-  ``masterCpuLimit``: Optional configuration. The CPU limits for the Determined master.

-  ``masterMemLimit``: Optional configuration. The memory limits for the Determined master.

-  ``taskContainerDefaults``: Specifies Docker defaults for all task containers. A task represents a
   single schedulable unit, such as a trial, command, or TensorBoard.

   -  ``networkMode``: The Docker network to use for the Determined task containers. If this is set
      to "host", Docker host-mode networking will be used instead. Defaults to "bridge".

   -  ``dtrainNetworkInterface``: The network interface to use during distributed training. If not
      set, Determined automatically determines the network interface. When training a model with
      multiple machines, the host network interface used by each machine must have the same
      interface name across machines. This is usually determined automatically, but there may be
      issues if there is an interface name common to all machines but it is not routable between
      machines. Determined already filters out common interfaces like ``lo`` and ``docker0``, but
      agent machines may have others. If interface detection is not finding the appropriate
      interface, the ``dtrainNetworkInterface`` option can be used to set it explicitly (e.g.,
      ``eth11``).

   -  ``forcePullImage``: Defines the default policy for forcibly pulling images from the Docker
      registry and bypassing the Docker cache. If a pull policy is specified in the :ref:`experiment
      config <exp-environment-image>` this default is overriden. Please note that as of November
      1st, 2020 unauthenticated users will be `capped at 100 pulls from Docker per 6 hours
      <https://www.docker.com/blog/scaling-docker-to-serve-millions-more-developers-network-egress/>`__.
      Defaults to ``false``.

   -  ``cpuPodSpec``: Sets the default pod spec for all non-GPU tasks. See :ref:`custom-pod-specs`
      for details.

   -  ``gpuPodSpec``: Sets the default pod spec for all GPU tasks. See :ref:`custom-pod-specs` for
      details.

   -  ``checkpointGcPodSpec``: Sets the custom pod spec for Checkpoint GC defaults. See
      :ref:`custom-pod-specs` for details.

   -  ``cpuImage``: Sets the default Docker image for all non-GPU tasks. If a Docker image is
      specified in the :ref:`experiment config <exp-environment-image>` this default is overriden.
      Defaults to: ``determinedai/pytorch-ngc-dev:0736b6d``.

   -  ``startupHook``: An optional inline script that will be executed as part of task set up.

   -  ``gpuImage``: Sets the default Docker image for all GPU tasks. If a Docker image is specified
      in the :ref:`experiment config <exp-environment-image>` this default is overriden. Defaults
      to: ``determinedai/pytorch-ngc-dev:0736b6d``.

   -  ``logPolicies``: Sets log policies for trials. For details, visit :ref:`log_policies
      <experiment-config-min-validation-period>`.

      .. code:: yaml

         logPolicies:
             - pattern: ".*uncorrectable ECC error encountered.*"
               action:
                 type: exclude_node

-  ``enterpriseEdition``: Specifies whether to use Determined enterprise edition.

-  ``imagePullSecretName``: Specifies the image pull secret for pulling the Determined master image.
   Required when using the enterprise edition.

-  ``masterService``: Specifies the configuration applied to the service for the Determined master.

   -  ``annotations``: Enables adding custom annotations to the Determined master service.

-  ``telemetry``: Specifies whether we collect anonymous information about the usage of Determined.

   -  ``enabled``: Whether collection is enabled. Defaults to ``true``.

-  ``observability``: Specifies whether Determined enables Prometheus monitoring routes. See
   :ref:`Prometheus <prometheus>` for details.

   -  ``enable_prometheus``: Whether Prometheus endpoints are present. Defaults to ``true``.

-  ``tensorboardTimeout``: Specifies the duration in seconds before idle TensorBoard instances are
   automatically terminated. A TensorBoard instance is considered to be idle if it does not receive
   any HTTP traffic. The default timeout is 300 (5 minutes).

-  ``initialUserPassword``: Specifies a string containing the default password for the admin and
   determined user accounts. (*Required*)

-  ``defaultPassword``: (*Deprecated*) Specifies a string containing the default password for the
   admin and determined user accounts. Use ``initialUserPassword`` instead.

-  ``logging``: Configures where trial logs are stored. This section takes the same shape as the
   logging configuration in the :ref:`cluster configuration <cluster-configuration>`, except that
   names are changed to camel case to match Helm conventions (e.g., ``skip_verify`` would be
   ``skipVerify`` here).

   -  ``logging.security.tls.certificate``: Contains the contents of an expected TLS certificate for
      the Elasticsearch cluster, rather than a path as it does in the cluster configuration. This
      can be conveniently set at the command line using ``helm install --set-file
      logging.security.tls.certificate=<path>``.

-  ``defaultScheduler``: Configures the default scheduler that Determined will use. Currently
   supports the ``coscheduler`` option, which enables the `lightweight coscheduling plugin
   <https://github.com/kubernetes-sigs/scheduler-plugins/tree/release-1.18/pkg/coscheduling>`__.
   Unless specified as ``coscheduler``, Determined will use the default Kubernetes scheduler.

-  ``resourcePools``: This section contains the names of the resource pools and their linked
   namespaces. Maps to the ``resource_pools`` section from the :ref:`master configuration
   <master-config-reference>`.

-  ``additional_resource_managers``: This section includes additional resource managers for
   launching jobs across multiple Kubernetes clusters. Maps to :ref:`additional_resource_managers
   <master-config-additional-resource-managers>` in the master configuration. An example
   configuration is provided in the ``values.yaml`` file.

   -  ``resource_manager``: Describes the configuration settings for the resource manager. Maps to
      :ref:`resource_manager <master-config-resource-manager>` in the master configuration.

      -  ``kubeconfig_secret_name``: Specifies the name of the secret containing the kubeconfig for
         the resource manager. This kubeconfig is used to connect to the Kubernetes cluster and
         launch tasks. Note that some kubeconfigs may require additional adjustments or
         modifications. For example some kubeconfigs reference file paths, which may need to be
         bind-mounted into the container or have their data paths encoded into the kubeconfig. Other
         kubeconfigs, like those for GKE, may require installing plugins into the Determined master
         container and binding certain credential files. (*Required*)

      -  ``kubeconfig_secret_value``: The name of the secret that contains the resource manager's
         kubeconfig. (*Required*)

   -  ``resource_pools``: The resource pool configuration. See :ref:`resource_pools
      <cluster-resource-pools>` for available configuration options.

-  ``retentionPolicy``: Specifies configuration settings for the retention of trial logs.

   -  ``logRetentionDays``: Specifies the number of days to retain logs for by default. Values
      should be between ``-1`` and ``32767``. The default value is ``-1``, retaining logs
      indefinitely. If set to ``0``, logs will be deleted during the next cleanup.

   -  ``schedule``: Specifies the schedule for cleaning up logs. Can be provided as a cron
      expression or a duration string.

.. include:: ../../_shared/note-dtrain-learn-more.txt
