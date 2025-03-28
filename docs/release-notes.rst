:orphan:

.. _release-notes:

###############
 Release Notes
###############

**************
 Version 0.38
**************

Version 0.38.1
==============

**Release Date:** March 19, 2025

**Security Fixes**

-  Dependency: Update SwaggerUI due to a potential phishing vulnerability behind the "Try-it-out"
   feature. Refer to the SwaggerUI security advisory `GHSA-qrmm-w75w-3wpx
   <https://github.com/swagger-api/swagger-ui/security/advisories/GHSA-qrmm-w75w-3wpx>`_.

-  Dependency: Update golang.org/x/crypto.

Version 0.38.0
==============

**Release Date:** November 22, 2024

**Breaking Changes**

-  ASHA: All experiments using ASHA hyperparameter search must now configure ``max_time`` and
   ``time_metric`` in the experiment config, instead of ``max_length``. Additionally, training code
   must report the configured ``time_metric`` in validation metrics. As a convenience, Determined
   training loops now automatically report ``batches`` and ``epochs`` with metrics, which you can
   use as your ``time_metric``. ASHA experiments without this modification will no longer run.

-  Custom Searchers: All custom searchers including DeepSpeed Autotune were deprecated in ``0.36.0``
   and are now being removed. Users are encouraged to use a preset searcher, which can be easily
   :ref:`configured <experiment-configuration_searcher>` for any experiment.

-  API: Custom Searcher (including DeepSpeed AutoTune) was deprecated in 0.36.0 and is now removed.
   We will maintain first-class support for a variety of preset searchers, which can be easily
   configured for any experiment. Visit :ref:`search-methods` for details.

**New Features**

-  API/CLI: Add support for access tokens. Add the ability to create and administer access tokens
   for users to authenticate in automated workflows. Users can define the lifespan of these tokens,
   making it easier to securely authenticate and run processes. Users can set global defaults and
   limits for the validity of access tokens by configuring ``default_lifespan_days`` and
   ``max_lifespan_days`` in the master configuration. Setting ``max_lifespan_days`` to ``-1``
   indicates an **infinite** lifespan for the access token. This feature enhances automation while
   maintaining strong security protocols by allowing tighter control over token usage and
   expiration. This feature requires Determined Enterprise Edition.

   -  CLI:

      -  ``det token create``: Create a new access token.
      -  ``det token login``: Sign in with an access token.
      -  ``det token edit``: Update an access token's description.
      -  ``det token list``: List all active access tokens, with options for displaying revoked
         tokens.
      -  ``det token describe``: Show details of specific access tokens.
      -  ``det token revoke``: Revoke an access token.

   -  API:

      -  ``POST /api/v1/tokens``: Create a new access token.
      -  ``GET /api/v1/tokens``: Retrieve a list of access tokens.
      -  ``PATCH /api/v1/tokens/{token_id}``: Edit an existing access token.

-  API: Introduce ``keras.DeterminedCallback``, a new high-level training API for TF Keras that
   integrates Keras training code with Determined through a single :ref:`Keras Callback
   <api-keras-ug>`.

-  API: Introduce ``deepspeed.Trainer``, a new high-level training API for DeepSpeedTrial that
   allows for Python-side training loop configurations and includes support for local training.

-  Cluster: In the enterprise edition of Determined, add :ref:`config policies <config-policies>` to
   enable administrators to set limits on how users can define workloads (e.g., experiments,
   notebooks, TensorBoards, shells, and commands). Administrators can define two types of
   configurations:

   -  **Invariant Configs for Experiments**: Settings applied to all experiments within a specific
      scope (global or workspace). Invariant configs for other tasks (e.g. notebooks, TensorBoards,
      shells, and commands) is not yet supported.

   -  **Constraints**: Restrictions that prevent users from exceeding resource limits within a
      scope. Constraints can be set independently for experiments and tasks.

-  Helm: Support configuring ``determined_master_host``, ``determined_master_port``, and
   ``determined_master_scheme``. These control how tasks address the Determined API server and are
   useful when installations span multiple Kubernetes clusters or there are proxies in between tasks
   and the master. Also, ``determined_master_host`` now defaults to the service host,
   ``<det_namespace>.<det_service_name>.svc.cluster.local``, instead of the service IP.

-  Helm: Add support for capturing and restoring snapshots of the database persistent volume. Visit
   :ref:`helm-config-reference` for more details.

-  New RBAC role: In the enterprise edition of Determined, add a ``TokenCreator`` RBAC role, which
   allows users to create, view, and revoke their own :ref:`access tokens <access-tokens>`. This
   role can only be assigned globally.

-  Experiments: Add a ``name`` field to ``log_policies``. When a log policy matches, its name shows
   as a label in the WebUI, making it easy to spot specific issues during a run. Labels appear in
   both the run table and run detail views.

   In addition, there is a new format: ``name`` is required, and ``action`` is now a plain string.
   For more details, refer to :ref:`log_policies <config-log-policies>`.

**Improvements**

-  Master Configuration: Add support for crypto system configuration for ssh connection.
   ``security.key_type`` now accepts ``RSA``, ``ECDSA`` or ``ED25519``. Default key type is changed
   from ``1024-bit RSA`` to ``ED25519``, since ``ED25519`` keys are faster and more secure than the
   old default, and ``ED25519`` is also the default key type for ``ssh-keygen``.

**Removed Features**

-  WebUI: "Continue Training" no longer supports configurable number of batches in the Web UI and
   will simply resume the trial from the last checkpoint.

**Known Issues**

-  PyTorch has `deprecated
   <https://pytorch.org/tutorials/intermediate/tensorboard_profiler_tutorial.html#use-tensorboard-to-view-results-and-analyze-model-performance>`
   their Profiler TensorBoard Plugin (``tb_plugin``), so some features may not be compatible with
   PyTorch 2.0 and above. Our current default environment image comes with PyTorch 2.3. If users are
   experiencing issues with this plugin, we suggest using an image with a PyTorch version earlier
   than 2.0.

**Bug Fixes**

-  Previously, during a grid search, if a hyperparameter contained an empty nested hyperparameter
   (that is, just an empty map), that hyperparameter would not appear in the hparams passed to the
   trial.

**Deprecations**

-  Experiment Config: The ``max_length`` field of the searcher configuration section has been
   deprecated for all experiments and searchers. Users are expected to configure the desired
   training length directly in training code.

-  Experiment Config: The ``optimizations`` config has been deprecated. Please see :ref:`Training
   APIs <apis-howto-overview>` to configure supported optimizations through training code directly.

-  Experiment Config: The ``scheduling_unit``, ``min_checkpoint_period``, and
   ``min_validation_period`` config fields have been deprecated. Instead, these configuration
   options should be specified in training code.

-  Experiment Config: The ``entrypoint`` field no longer accepts ``model_def:TrialClass`` as trial
   definitions. Please invoke your training script directly (``python3 train.py``).

-  Core API: The ``SearcherContext`` (``core.searcher``) has been deprecated. Training code no
   longer requires ``core.searcher.operations`` to run, and progress should be reported through
   ``core.train.report_progress``.

-  DeepSpeed: The ``num_micro_batches_per_slot`` and ``train_micro_batch_size_per_gpu`` attributes
   on ``DeepSpeedContext`` have been replaced with ``get_train_micro_batch_size_per_gpu()`` and
   ``get_num_micro_batches_per_slot()``.

-  Horovod: The Horovod distributed training backend has been deprecated. Users are encouraged to
   migrate to the native distributed backend of their training framework (``torch.distributed`` or
   ``tf.distribute``).

-  Trial APIs: ``TFKerasTrial`` has been deprecated. Users are encouraged to migrate to the new
   :ref:`Keras Callback <api-keras-ug>`.

-  Launchers: The ``--trial`` argument in Determined launchers has been deprecated. Please invoke
   your training script directly.

-  ASHA: The ``stop_once`` field of the ``searcher`` config for ASHA searchers has been deprecated.
   All ASHA searches are now early-stopping based (``stop_once: true``) instead of promotion based.

-  CLI: The ``--test`` and ``--local`` flags for ``det experiment create`` have been deprecated. All
   training APIs now support local execution (``python3 train.py``). Please see ``training apis``
   for details specific to your framework.

-  Web UI: Previously, trials that reported an ``epoch`` metric enabled an epoch X-axis in the Web
   UI metrics tab. This metric name has been changed to ``epochs``, with ``epoch`` as a fallback
   option.

-  Database: After Amazon Aurora V1 reaches End of Life, support for Amazon Aurora V1 in ``det
   deploy aws`` will be removed. Future deployments will default to the ``simple-rds`` type, which
   uses Amazon RDS for PostgreSQL. We recommend that users migrate to Amazon RDS for PostgreSQL. For
   more information, visit the `migration instructions
   <https://gist.github.com/maxrussell/c67f4f7d586d55c4eb2658cc2dd1c290>`_.

-  Database: As a follow-up to the earlier notice, PostgreSQL 12 will reach End of Life on November
   14, 2024. Instances still using PostgreSQL 12 or earlier should upgrade to PostgreSQL 13 or later
   to maintain compatibility. The application will log a warning if it detects a connection to any
   PostgreSQL version older than 12, and this warning will be updated to include PostgreSQL 12 once
   it is End of Life.

**************
 Version 0.37
**************

Version 0.37.0
==============

**Release Date:** September 30, 2024

**Breaking Changes**

-  API: Remove the ``model_hub`` library from Determined.

-  Starting with this release, ``MMDetTrial`` and ``BaseTransformerTrial`` are removed. HuggingFace
   users should refer to the provided `HuggingFace TrainerAPI examples
   <https://github.com/determined-ai/determined/tree/main/examples/hf_trainer_api>`__, which use a
   custom callback instead of BaseTransformerTrial. Users of ``MMDetTrial`` can refer to :ref:`Core
   API <api-core-ug>`.

**New Features**

-  Webhooks: Add support for experiment monitoring and alerting. Capabilities include
   workspace-level subscriptions for "All experiments" or "Specific experiment(s) with matching
   configuration" options. New trigger types include ``COMPLETED``, ``ERROR``, ``TASKLOG``, and
   ``CUSTOM``. Support for custom triggers, code-based alerts, experiment-specific webhook
   exclusions, and editable webhook URLs is also added. For details, visit
   :ref:`supported-webhook-triggers`.

-  Master Configuration: Add support for POSIX claims in the master configuration. It now accepts
   ``agent_uid_attribute_name``, ``agent_gid_attribute_name``, ``agent_user_name_attribute_name``,
   or ``agent_group_name_attribute_name``. Refer to the :ref:`OIDC master configuration
   <master-config-oidc>` or :ref:`SAML master configuration <master-config-saml>` for details. If
   any of these fields are configured, they will sync with the database.

**Improvements**

-  WebUI: Change the "Compute Slots Allocated" label to "Unspecified Slots Allocated" for resource
   pools with no or multiple slot types. Add error logs for zero or multi-slot-type cases and update
   the progress bar to include all agents when the slot type is ``TYPE_UNSPECIFIED``.

**Bug Fixes**

-  API/Tasks: Fix a bug where a master-configured ``log_retention_days`` value is not applied to
   experiments and tasks. The master-configured value is now correctly applied to new experiments,
   and all pre-existing experiments will also follow the specified ``log_retention_days``.

**************
 Version 0.36
**************

Version 0.36.0
==============

**Release Date:** August 23, 2024

**New Features**

-  WebUI: In the enterprise edition of Determined, when RBAC is enabled, allow Viewer, Editor,
   GenAI, and Workspace Admin roles to view resource quotas for each workspace in the WebUI. When
   RBAC is not enabled, any user can view resource quotas.

-  RBAC: Add a pre-canned role called ``EditorProjectRestricted`` that supersedes the ``Viewer``
   role and precedes the ``Editor`` role.

   -  Like the ``Editor`` role, the ``EditorProjectRestricted`` role grants the permissions to read,
      create, edit, or delete experiments and NTSC (Notebook, Tensorboard, Shell or Command) type
      workloads within its scope. However, the ``EditorProjectRestricted`` role lacks the
      permissions to create or update projects.

-  Kubernetes: Add experimental support for AMD ROCm GPUs. To use, set ``slotType=rocm``. Visit
   :ref:`helm-config-reference` for more details.

-  Images: Add New ROCm 6.1 images with DeepSpeed for MI300x users. Dev versions of these images can
   be found in our Docker Hub, under `pytorch-infinityhub-dev
   <https://hub.docker.com/repository/docker/determinedai/pytorch-infinityhub-dev/tags>`__ and
   `pytorch-infinityhub-hpc-dev
   <https://hub.docker.com/repository/docker/determinedai/pytorch-infinityhub-hpc-dev/tags>`__.
   Users can build these images locally based on the Dockerfiles found in our `environments
   repository
   <https://github.com/determined-ai/environments/blob/main/Dockerfile-infinityhub-pytorch>`__.

-  Master: Add a ``ui_customization`` option to the :ref:`master configuration
   <master-config-reference>` for specifying a custom logo for the WebUI.

**Bug Fixes**

-  Experiments: Report an experiment's status as FAILED if any failure occurs during the shutdown
   process, before the experiment has completed gracefully.

**Deprecations**

-  Custom Searchers: All custom searchers including DeepSpeed Autotune have been deprecated. This
   feature will be removed in a future release. We will maintain first-class support for a variety
   of preset searchers, which can be easily configured for any experiment. Visit
   :ref:`search-methods` for details.

-  Cluster: Amazon Aurora V1 will reach End of Life at the end of 2024 and will no longer be the
   default persistent storage for AWS Determined deployments. Users should migrate to Amazon RDS for
   PostgreSQL.

-  Cluster: After Amazon Aurora V1 reaches End of Life, support for Amazon Aurora V1 in ``det deploy
   aws`` will be removed. The deployment will default to the ``simple-rds`` type, which uses Amazon
   RDS.

-  Database: Postgres 12 will reach End of Life on November 14, 2024. Determined instances using
   Postgres 12 or earlier should upgrade to Postgres 13 or later to ensure continued support.

-  Kubernetes Scheduling: Support for the priority with preemption scheduler for Kubernetes Resource
   Managers was deprecated in 0.35.0 and is now removed. Users should transition to the default
   scheduler. Visit :ref:`Kubernetes Default Scheduler <kubernetes-default-scheduler>` for details.

**************
 Version 0.35
**************

Version 0.35.0
==============

**Release Date:** August 08, 2024

**Breaking Changes**

-  Master Configuration: Replace ``resource_manager.name`` with ``resource_manager.cluster_name``
   for better clarity and to support multiple resource managers.

   -  Resource managers operate relative to the specific cluster providing resources for Determined
      tasks, so changing the cluster will affect the resource manager's responses.

   -  The ``cluster_name`` must be unique for all resource managers when deploying multiple resource
      managers in Determined.

   -  When upgrading, specify ``resourceManager.clusterName`` in your ``values.yaml`` to override
      ``resource_manager.name`` and/or remove the ``name`` field from your ``resource_manager``
      config altogether.

   -  For additional resource managers, you must change ``additional_resource_manager[i].name`` to
      ``additional_resource_manager[i].cluster_name`` in your ``values.yaml``.

-  Master Configuration: Replace ``resource_manager.namespace`` with
   ``resource_manager.default_namespace``.

   -  The namespace field in the Kubernetes Resource Manager configuration is no longer supported
      and is replaced by ``default_namespace``.

   -  This field serves as the default namespace for deploying namespaced resources when the
      workspace associated with a workload is not bound to a specific namespace.

   -  If unset, the workloads will be sent to the release namespace during determined helm installs
      or upgrades and will be sent to the default Kubernetes namespace, "default", during non-helm
      determined deployments.

-  Tasks: The :ref:`historical usage <historical-cluster-usage-data>` CSV file has been updated. The
   header row for slot-hours is now named ``slot_hours`` instead of ``gpu_hours`` to accurately
   reflect the allocation time for resource pools including those without GPUs. In addition, a new
   column, ``resource_pool``, has been added to provide the resource pool for each allocation.

-  Cluster: The ``kubernetes_namespace`` field in the resource pool configuration is no longer
   supported. Users can now submit workloads to specific namespaces by binding workspaces to
   namespaces using the CLI, WebUI, or API.

-  Cluster: The ``resources.agent_label`` task option and the ``label`` option in the agent config
   have been removed. Beginning with 0.20.0 release, these options have been ignored. Please remove
   any remaining references from configuration files and use ``resource_pool`` instead.

**New Features**

-  WebUI/CLI/API: Allow admins to bind namespaces to workspaces and manage resource quotas for
   auto-created namespaces directly.

   -  WebUI: Add a "Namespace Bindings" section to the Create and Edit Workspace modals.

      -  Users can input a namespace for a Kubernetes cluster. If no namespace is specified, the
         workspace will be bound to the ``resource_manager.default_namespace`` field in the master
         configuration YAML or the "default" Kubernetes namespace.

      -  In the enterprise edition, users can auto-create namespaces and set resource quotas,
         limiting GPU requests for that workspace. The Edit Workspace modal displays the lowest GPU
         limit resource quota within the bound namespace.

      -  Once saved, all workloads in the workspace will be sent to the bound namespace. Changing
         the binding will affect future workloads, while in-progress workloads remain in their
         original namespace.

      -  For help with workspace-namespace bindings, visit :ref:`Manage Workspace-Namespace Bindings
         <k8s-resource-caps>`.

   -  CLI: Add new commands for creating and managing workspace namespace bindings.

      -  Allow creating namespace bindings during workspace creation with ``det w create
         <workspace-id> --namespace <namespace-name>`` or later with ``det w bindings set
         <workspace-id> --namespace <namespace-name>``.

      -  In the enterprise edition, users can use additional arguments ``--auto-create-namespace``
         and ``--auto-create-namespace-all-clusters`` to bind workspaces to auto-created namespaces.
         Users can set resource quotas during workspace creation with ``det w create
         <workspace-name> --cluster-name <cluster-name> --auto-create-namespace --resource-quota
         <resource-quota>``, or later with ``det w resource-quota set <workspace-id> <quota>
         --cluster-name <cluster-name>`` if their workspace is bound to an auto-created namespace.

      -  Add a command to delete namespace bindings with ``det w bindings delete <workspace-id>
         --cluster-name <cluster-name>``.

      -  Add a command to list bindings for a workspace with ``det w bindings list
         <workspace-name>``.

      -  The ``--cluster-name`` field is required only for MultiRM setups when
         ``--auto-create-namespace-all-clusters`` is omitted.

   -  API: Add new endpoints for creating and managing workspace namespace bindings.

      -  Add POST and DELETE endpoints to ``/api/v1/workspaces/{workspace_id}/namespace-bindings``
         for setting and deleting workspace namespace bindings.
      -  Add a GET endpoint ``/api/v1/workspaces/{id}/list-namespace-bindings`` to list namespace
         bindings for a workspace.
      -  Add a POST endpoint ``/api/v1/workspaces/{id}/set-resource-quota`` to set resource quotas
         on workspaces bound to auto-created namespaces.
      -  Add a GET endpoint ``/api/v1/workspaces/{id}/get-k8s-resource-quotas`` to retrieve enforced
         Kubernetes GPU resource quotas for workspace bound namespaces.

-  WebUI: Enable users to add or remove hyperparameters during hyperparameter searches.

-  WebUI: Experiments with configured Pachyderm data integration now display a link to the Pachyderm
   repo in the trial view page. The link is also available when viewing checkpoints derived from the
   Pachyderm data. For a preview, visit: :ref:`Pachyderm <pachyderm-integration>` data lineage.

-  WebUI: In the Experimental features, Flat Runs View is now "on" by default in the :ref:`WebUI
   <web-ui-if>`. Users can still toggle this feature "off". This update improves the ability to
   compare model performance between different trials, based on user feedback that most Determined
   users run single-trial experiments.

   -  "Experiments" are now called "searches" and "trials" are now called "runs" for better clarity.
   -  The "experiment list" is now called the "run list", showing all trials from experiments in the
      project. It functions similarly to the previous new experiment list.
   -  Multi-trial experiments can be viewed in the new searches view, which allows for sorting,
      filtering and navigating multi-trial experiments.
   -  When viewing a multi-trial experiment, a list of trials is displayed, allowing for sorting,
      filtering and arbitrary comparison between trials.

-  WebUI: Add resource allocation information to the trial details page.

-  WebUI: Allow users to continue a canceled or errored multi-trial experiment for searcher type
   ``random`` or ``grid``.

-  Master Configuration: Add an ``always_redirect`` option to OIDC and SAML configurations. When
   enabled, this option bypasses the standard Determined sign-in page and routes users directly to
   the configured SSO provider. This redirection persists unless the user explicitly signs out
   within the WebUI.

-  Experiments: Obfuscate subfields of ``data.secrets`` in the :ref:`experiment configuration
   <experiment-config-data>`.

-  CLI: Add a new command, ``det cmd describe COMMAND_ID`` to allow users to fetch the metadata of a
   single command.

**Improvements**

-  Switch the default AWS instance type from ``m5.large`` to ``m6i.large``. This change enhances
   performance without affecting the cost.
-  WebUI: In the enterprise edition, redirect SSO users to the SSO provider's authentication URIs
   when their session token has expired, instead of displaying the Determined sign-in page.

**Bug Fixes**

-  WebUI: Fix a bug where the Compare view on the Project Details page did not allow comparison of
   experiments selected from different pages.
-  WebUI: Fix endless metrics fetching in "Visualization" tab in experiment details page for
   cancelled experiments that do not have metrics.
-  Fix two places where aggregated queued stats could have shown inflated values. The total queued
   aggregated time and today's queued aggregated time calculations were both affected.
-  CLI: Fix an error related to ``det cmd list --csv``
-  WebUI: Fix missing data in Historic Usage Charts due to erroneous date parsing.

**Deprecations**

-  Detached mode: The ``defaults`` and ``unmanaged`` parameters of the ``init`` function for
   unmanaged experiment have been deprecated and will be removed in a future version. Please use
   ``config`` instead.

-  Agent and Kubernetes Resource Manager: Jobs can no longer be moved within the same priority
   group. To reposition a job, update its priority using the CLI or WebUI. For detailed
   instructions, visit :ref:`modify-job-queue-cli`. This change was announced in version 0.33.0.

-  AgentRM: Support for Singularity, Podman, and Apptainer was deprecated in 0.33.0 and is now
   removed. Docker is the only container runtime supported by Agent resource manager (AgentRM). It
   is still possible to use podman with AgentRM by using the podman emulation layer. For detailed
   instructions, visit: `Emulating Docker CLI with Podman
   <https://podman-desktop.io/docs/migrating-from-docker/emulating-docker-cli-with-podman>`. You
   might need to also configure ``checkpoint_storage`` in experiment or master configurations. In
   the enterprise edition, Slurm resource manager still supports Singularity, Podman, or Apptainer
   use.

-  Kubernetes Scheduling: Support for the priority scheduler for Kubernetes Resource Managers is
   discontinued and may be removed in a future release due to limited usage. Users should transition
   to the default scheduler. Visit :ref:`Kubernetes Default Scheduler
   <kubernetes-default-scheduler>` for details.

-  API: The ``model_hub`` library is now deprecated. Users of MMDetTrial and BaseTransformerTrial
   should switch to :ref:`Core API <api-core-ug>` or the :ref:`PyTorch Trainer <pytorch_trainer_ug>`
   for integrations with ``mmcv`` and ``huggingface``.

**************
 Version 0.34
**************

Version 0.34.0
==============

**Release Date:** June 28, 2024

**Breaking Changes**

-  Images: The default environment includes images that support PyTorch. Therefore, TensorFlow users.
      must configure their experiments to target our non-default TensorFlow images. Details on this
      process can be found at :ref:`set-environment-images`.

-  Images: Our new default images are based on Nvidia NGC. While we provide a recommended NGC
   version, users can build their own images using any NGC version that meets their specific
   requirements. For more information, visit :ref:`ngc-version`

**New Features**

-  Kubernetes: The system now launches Kubernetes jobs on behalf of users when they submit workloads
   to Determined, instead of launching Kubernetes pods. This change allows Determined to work
   properly with other Kubernetes features like resource quotas.

   As a result, permissions are now required to create, get, list, delete, and watch Kubernetes job
   resources.

-  WebUI: Add the ability for administrators to use the CLI to set a message to be displayed on all
   pages of the WebUI (for example, ``det master cluster-message set -m "Your message"``). Optional
   flags are available for scheduling the message with a start time and an end time. Administrators
   can clear the message anytime using ``det master cluster-message clear``. Only one message can be
   active at a time, so setting a new message will replace the previous one.

-  Kubernetes: Add a feature where Determined offers the users to provide custom Checkpoint GC pod spec.
      This configuration is done using the ``task_container_defaults.checkpointGcPodSpec`` field
      within your ``value.yaml`` file. User can create a custom pod specification for CheckpointGC,
      it will override the default experiment's pod spec settings. Determined by default uses the
      experiment's pod spec, but by providing custom pod spec users have the flexibility to
      customize and configure the pod spec directly in this field. User can tailor the garbage
      collection settings according to the specific GC needs.

-  Kubernetes: The :ref:`Internal Task Gateway <internal-task-gateway>` feature enables Determined
   tasks running on remote Kubernetes clusters to be exposed to the Determined master and proxies.
   This feature facilitates multi-resource manager setups by configuring a Gateway controller in the
   external Kubernetes cluster.

.. important::

   Enabling this feature exposes Determined tasks to the outside world. It is crucial to implement
   appropriate security measures to restrict access to exposed tasks and secure communication
   between the external cluster and the main cluster. Recommended measures include:

      -  Setting up a firewall
      -  Using a VPN
      -  Implementing IP whitelisting
      -  Configuring Kubernetes Network Policies
      -  Employing other security measures as needed

-  Kubernetes Configuration: Allow Cluster administrators to define Determined resource pools on
   Kubernetes using node selectors and/or affinities. Configure these settings at the default pod
   spec level under ``task_container_defaults.cpu_pod_spec`` or
   ``task_container_defaults.gpu_pod_spec``. This allows a single cluster to be divided into
   multiple resource pools using node labels.

-  WebUI: Allow resource pool slot counts to reflect the state of the entire cluster. Allow slot
   counts and scheduling to respect node selectors and affinities. This impacts Determined clusters
   deployed on Kubernetes with multiple resource pools defined in terms of node selectors and/or
   affinities.

**Bug Fixes**

-  Kubernetes: Fix an issue where where jobs would remain in "QUEUED" state until all pods were
   running. Jobs will now correctly show as "SCHEDULED" once all pods have been assigned to nodes.
-  Notebooks: Fix an issue introduced in 0.30.0 where idle notebooks were not terminated as
   expected.

**Security Fixes**

   -  CLI: When deploying locally using ``det deploy local`` with ``master-up`` or ``cluster-up``
      commands and no user accounts have been created yet, an initial password will be automatically
      generated and shown to the user (with the option to change it) if neither
      ``security.initial_user_password`` in ``master.yaml`` nor the ``--initial-user-password`` CLI
      flag is present.

**Deprecations**

-  Agent Resource Manager: Round robin scheduler is removed for Agent Resource Managers. Deprecation
   was announced in release 0.33.0. Users should transition to priority scheduler.
-  Machine Architectures: Support for PPC64/POWER builds for all environments has been deprecated
   and is now being removed. Users should transition to ARM64/AMD64.

**************
 Version 0.33
**************

Version 0.33.0
==============

**Release Date:** May 29, 2024

**Breaking Changes**

-  Helm: An entry for ``initialUserPassword`` is now required when running ``helm install``.
   Existing deployments are unaffected. See :ref:`Helm Chart <helm-config-reference>`.

-  Web UI: Enforce password requirements for all new non-remote users. See
   :ref:`password-requirements` for details.

   -  Applies to users created using the **Add User** button in the Web UI for admins.
   -  Admins can change the passwords of other users using the same interface.
   -  Does not affect existing users with empty or non-compliant passwords, but setting strong
      passwords for these users is recommended.

**Improvements**

Kubernetes: Add Determined resource information such as ``workspace`` and ``task ID`` as pod labels.
This improvement facilitates better resource tracking and management within Kubernetes environments.

Configuration: Introduce a DCGM Helm chart and Prometheus configuration to the
``tools/observability`` directory. Additionally, two new dashboards, "API Monitoring" and "Resource
Utilization", have been added to improve observability and operational insight. Visit `Kubernetes
Observability <https://docs.determined.ai/latest/integrations/observability/_index.html>`__ for a
complete setup guide.

-  WebUI: Allow users to create and manage configuration templates through the WebUI.
-  Commands: Commands now support automatically executing a ``startup-hook.sh`` script if it is
   present in the command's context directory.

**Bug Fixes**

-  Kubernetes: Fix an issue where Determined failed to report slots as occupied when non Determined
   jobs were running on namespaces besides 'default'. For Determined to detect non Determined jobs
   they must be running in a namespace that Determined can launch jobs into.

-  Kubernetes: Fix an issue where the cluster page displayed slots out of order on refresh. Slots
   are now consistently filled from left to right, even with more than 10 GPUs and when using RBAC.

**Deprecations**

To enhance stability and streamline the onboarding process, we may remove the following features in
future releases. Our goal is for Agent Resource Manager environments to function seamlessly
out-of-the-box with minimal customization required.

Agent Resource Manager:

-  Container Runtimes: Due to limited usage, we will limit supported container runtimes to Docker
   for the Agent Resource Manager. This does not impact Kubernetes, Slurm or PBS environments.

-  Job Scheduling: The default scheduler is now ``priority``. Support for round-robin and fair share
   schedulers has been discontinued. We recommend using the priority scheduler, as it meets most
   scheduling needs and simplifies configuration. To move a job, you will need to adjust its
   priority; jobs cannot be shifted within the same priority group.

-  AMD GPUs: Due to limited usage, we will limit supported accelerators to NVIDIA GPUs. If you have
   a use case requiring AMD GPU support with the Agent Resource Manager, please reach out to us via
   a `GitHub Issue <https://github.com/determined-ai/determined/issues>`__ or `community slack
   <https://determined-community.slack.com/join/shared_invite/zt-1f4hj60z5-JMHb~wSr2xksLZVBN61g_Q>`__!
   This does not impact Kubernetes or Slurm environments.

Machine Architectures: PPC64/POWER builds across all environments are no longer supported.

**************
 Version 0.32
**************

Version 0.32.1
==============

**Release Date**: May 10, 2024

**Bug Fixes**

-  Kubernetes: Fix an issue introduced in 0.32.0 where workspaces with names incompatible with
   Kubernetes naming requirements would cause jobs in that workspace to fail.

Version 0.32.0
==============

**Release Date:** May 08, 2024

Notice: This release contains an important fix for a bug that poses data loss risk when using the
Experiment table in the project view in the WebUI. All users on affected versions are strongly
encouraged to upgrade as soon as possible. For more details, scroll down to ``Bug Fixes``.

**Breaking Changes**

-  Python SDK and CLI: Password requirements are now enforced for all non-remote users. (The
   requirements do not apply to remote users, since they use single sign-on.) Existing users with
   empty or non-compliant passwords can still sign in. However, we recommend updating these
   passwords to meet the new requirements as soon as possible. For more information, visit
   :ref:`password-requirements`.

   This change affects the :meth:`~determined.experimental.client.Determined.create_user` and
   :meth:`~determined.experimental.client.User.change_password` SDK methods and the ``det user
   create`` and ``det user change-password`` CLI commands.

   When creating non-remote users at the CLI with ``det user create``, setting a password is now
   mandatory. You can set the password interactively by following the prompts during user creation
   or non-interactively with the ``--password`` option.

**New Features**

-  Kubernetes: In the enterprise edition, add the ability to set up the Determined master service on
   one Kubernetes cluster and manage workloads across different Kubernetes clusters. Additional
   non-default resource managers and resource pools are configured under the
   ``additional_resource_managers`` section (additional resource managers are required to have at
   least one resource pool defined). Additional resource managers and their resource pools must have
   unique names. For more information, visit :ref:`master configuration <master-config-reference>`.
   Support for notebooks and other workloads that require proxying is under development.

-  API/CLI/WebUI: In the enterprise edition, route any requests to resource pools not defined in the
   master configuration to the default resource manager, not any additional resource manager, if
   defined.

-  Configuration: In the enterprise edition, add an ``additional_resource_managers`` section that
   can define multiple resource managers following the same patteroas ``resource_manager``. Add
   ``name`` and ``metadata`` fields to individual resource manager definitions.

-  WebUI: In the enterprise edition, add the ability to view resource manager name for resource
   pools.

**Improvements**

-  Configuration: The master configuration parameter ``observability.enable_prometheus`` now
   defaults to ``true``. Consequently, Prometheus endpoints are enabled by default, which does not
   affect clusters that do not use Prometheus.

-  Experiment metrics tracking: Add enhanced support for metrics with long names. Previously,
   metrics with names exceeding 63 characters were recorded but not displayed in the UI or returned
   via APIs.

**Bug Fixes**

-  A bug was fixed impacting the selection functionality in the Experiments page. From version
   0.27.1 to version 0.31.0, this bug was causing actions to be applied to more experiments than are
   visibly selected. For example, when using the **Select All > Actions > Move** sequence to
   transfer all experiments from one project to another, the action may inadvertently include
   experiments not only from the targeted project but also from other projects you have permissions
   to edit. We urge all users on the affected versions to upgrade as soon as possible. The following
   applies to versions 0.27.1 to 0.31.0:

   -  There is a risk of data loss if, when attempting to delete a set of experiments, the action
      inadvertently deletes a larger set than intended.

   -  When role-based access control (an enterprise edition feature) is enabled, there is a risk of
      a permissions leak if moving experiments from one project to another inadvertently includes
      experiments from other workspaces.

   -  This issue affects all bulk actions including delete, move, archive, unarchive, resume, pause,
      kill, stop, and view in TensorBoard.

   -  We strongly advise refraining from using the experiment table in the project view to take any
      actions.

   -  Workaround: To manage actions on a single trial, use the trial view in the WebUI.
      Alternatively, for bulk actions affected by this issue, consider using the command-line
      interface (CLI). You can also turn off the New Experiment List setting under the User Settings
      > Experimental section. For more information visit Manage User Settings under :ref:`WebUI
      <web-ui-if>`.

-  A bug was fixed impacting deployments using Amazon Aurora PostgreSQL-Compatible Edition
   Serverless V1 as the database. Since version 0.28.1, deployments using Amazon Aurora
   PostgreSQL-Compatible Edition Serverless V1 as the database have been at risk of becoming
   unresponsive due to certain autoscaling errors. This issue affects multiple ``det deploy aws``
   deployment types, including ``simple``, ``vpc``, ``efs``, ``fsx``, and ``secure``. Installations
   using AWS RDS, including ``det deploy aws --deployment-type=govcloud``, are not affected. We urge
   all users with affected setups to upgrade as soon as possible.

**************
 Version 0.31
**************

Version 0.31.0
==============

**Release Date:** April 17, 2024

**Breaking Changes**

-  SAML: The underlying SAML implementation has been updated to use a newer, more maintained
   library. As a result, the master config no longer accepts the ``idp_cert_path`` field and now
   requires the ``idp_metadata_url`` field when using SAML.

**New Features**

-  API: Add a new API endpoint, ``/health``, that provides information about the status of
   Determined's connections to the database, Kubernetes API server, and Slurm launcher integration.

   Visit the :ref:`rest-api` documentation for more information about this endpoint.

-  Logging: Add a ``retention_policy`` section to the master config file for specifying the default
   log retention policy. Experiments can override the default log retention settings with the
   ``retention_policy.log_retention_days`` config option. See :ref:`master-config-reference` and
   :ref:`experiment-config-reference` for more details.

-  CLI: Add commands, ``det e set log-retention <exp-id>`` and ``det t set log-retention
   <trial-id>``, to allow the user to set the length of log retention for experiments and trials.
   Both commands can specify a length in days with the arguments ``--days <number of days>``. The
   number of days must be between -1 and 32767, where -1 retains logs forever. ``--forever`` is
   equivalent to ``--days -1``. Add ``det task cleanup-logs`` command to allow the administrators to
   manually initiate log retention cleanup.

-  WebUI: Add support for retaining logs for multiple experiments by selecting experiments from the
   experiment list page and choosing **Retain Logs** from **Actions**. Users can then input the
   desired number of days for log retention or select the "Forever" checkbox for indefinite log
   retention. The number of days must be between -1 and 32767, where -1 retains logs forever.

   There is a new column on the trial list page, "Log Retention Days", that displays the number of
   days for which logs will be retained for each trial after creation.

-  Master config: Add a new field to task container defaults named ``startup_hook`` that allows for
   the specification of an inline script to be executed after task setup.

**Improvements**

-  CLI: The ``--add-tag`` flag to ``det deploy aws up`` will now apply tags to dynamic agents
   launched.

**Bug Fixes**

-  API: Fix a bug where calling ``det job update`` could prevent jobs from being scheduled and cause
   ``det job ls`` to hang.

**Security Fixes**

-  Helm: When deploying a new cluster with Helm, configuring an initial password for the "admin" and
   "determined" users is required and is no longer a separate step. To specify an initial password
   for these users, set either ``initialUserPassword`` (preferred) or ``defaultPassword``
   (deprecated) in the ``helm/charts/determined/values.yaml`` file. For reference, see
   :ref:`helm-config-reference`.

**************
 Version 0.30
**************

Version 0.30.0
==============

**Release Date:** April 04, 2024

**Breaking Changes**

-  API: :class:`~determined.pytorch.Trainer` no longer supports the ``Trainer.configure_profiler``
   option. Profiling is now enabled through the ``Trainer.fit(profiling_enabled=True)`` call.

-  Database migration: System metrics collected by the Determined profiler are now stored in the
   generic ``metrics`` table. This requires a few schema changes to the ``metrics`` table that will
   be run during migrations.

   .. important::

      This migration may take more time for deployments with a large amount of stored metrics.

**New Features**

-  Core API: The Determined profiler is now accessible from the Core API. It collects system
   metrics, which can be viewed in the WebUI under the experiment's "Profiler" tab. See the
   :ref:`Core API guide <core-profiler>` for details.

**Removed Features**

-  Profiler: Support for timing metrics and related configurations has been removed. The Determined
   profiler now only collects system metrics and defers to our native profiler integrations for
   training-specific profiling. Users are encouraged to configure profilers native to their
   :ref:`training API <apis-howto-overview>` for this functionality.

   -  Historical data for timing metrics is retained in the ``trial_profiler_metrics`` database
      table, but they are no longer being collected or rendered in the WebUI.

   -  Historical data for system metrics generated by trials before this release are not
      automatically migrated due to time cost. For users wanting to view historical system metrics
      in the WebUI, we provide an `optional migration script
      <https://github.com/determined-ai/determined/blob/main/master/static/optional_migrations/20240325144732_trial-profiler-metrics-migration.tx.up.sql>`__
      that can be run manually.

   -  Configuration: The ``timings_enabled``, ``begin_on_batch``, and ``end_after_batch`` options in
      the ``profiling`` section of experiment configurations are no longer supported.

**************
 Version 0.29
**************

Version 0.29.1
==============

**Release Date:** March 18, 2024

**New Features**

-  Include early-access NVIDIA NGC-based images in our environment offerings. These images are
   accessible from `pytorch-ngc <https://hub.docker.com/r/determinedai/pytorch-ngc>`__ or
   `tensorflow-ngc <https://hub.docker.com/r/determinedai/tensorflow-ngc>`__. By downloading and
   using these images, users acknowledge and agree to the terms and conditions of all third-party
   software licenses contained within, including the `NVIDIA Deep Learning Container License
   <https://developer.download.nvidia.com/licenses/NVIDIA_Deep_Learning_Container_License.pdf>`__.
   Users can build their own images from a specified NGC container version using the
   ``build-pytorch-ngc`` or ``build-tensorflow-ngc`` targets in the makefile in our `environments
   repository <https://github.com/determined-ai/environments>`__.

-  RBAC: Add a pre-canned role called ``EditorRestricted`` which supersedes the ``Viewer`` role and
   precedes the ``Editor`` role.

   -  Like the ``Editor`` role, the ``EditorRestricted`` role grants the permissions to create,
      edit, or delete projects and experiments within its designated scope. However, the
      ``EditorRestricted`` role lacks the permissions to create or update NSC (Notebook, Shell or
      Command) type workloads.

      Therefore, a user with ``EditorRestricted`` privileges in a given scope is limited when using
      the WebUI within that scope since the option to launch JupyterLab notebooks and kill running
      tasks will be unavailable. The user will also be unable to run CLI commands that create scoped
      notebooks, shells, and commands and will be unable to perform updates on these tasks (such as
      changing the task's priority or deleting it). ``EditorRestricted`` users can still open and
      use scoped JupyterLab notebooks and perform all experiment-related jobs, just like those with
      the ``Editor`` role.

   -  The ``EditorRestricted`` role allows workspace and cluster editors and admins to have more
      fine-grained control over GPU resources. Thus, users with this role lack the ability to launch
      or modify tasks that indefinitely consume slot-requesting resources within a given scope.

**Improvements**

-  Images: Eliminate TensorFlow 2.8 images from our offerings. Default TensorFlow 2.11 images remain
   available for TensorFlow users.

**Bug Fixes**

-  Experiments: Fix an issue where experiments in the ``STOPPING_CANCELED`` state on master restart
   would leave unkillable containers running on agents.

Version 0.29.0
==============

**Release Date:** March 05, 2024

**Breaking Changes**

-  Add a new requirement for runtime configurations that there be a writable ``$HOME`` directory in
   every container. Previously, there was limited support for containers without a writable
   ``$HOME``, merely by coincidence. This change could impact users in scenarios where jobs were
   configured to run as the ``nobody`` user inside a container, instead of the ``det-nobody``
   alternative recommended in :ref:`run-unprivileged-tasks`. Users combining non-root tasks with
   custom images not based on Determined's official images may also be affected. Overall, it is
   expected that few or no users are affected by this change.

**Removed Features**

-  Removed the accidentally exposed ``Session`` object from the ``det.experimental.client``
   namespace. It was never meant to be a public API and it was not documented in :ref:`python-sdk`,
   but was nonetheless exposed in that namespace. It was also available as a deprecated legacy
   alias, ``det.experimental.Session``. It is expected that most users use the Python SDK normally
   and are unaffected by this change, since the ``det.experimental.client``'s ``login()`` and
   ``Determined()`` are unaffected.

**Improvements**

-  Configure log settings for the Determined agent in the configuration file used to launch
   Determined clusters by setting ``log.level`` and ``log.color`` appropriately.

**Bug Fixes**

-  Resource Manager: Prevent connections from duplicate agents. Agent connection attempts will be
   rejected if there's already an active connection from a matching agent ID. This prevents and
   replaces previous behavior of stopping the running agent when a duplicate connection attempt is
   made (causing both connections to fail).

**Security**

-  Add a configuration setting, ``initial_user_password``, to the master configuration file forcing
   the setup of an initial user password for the built-in ``determined`` and ``admin`` users during
   the first launch, when a cluster's database is bootstrapped.

.. important::

   For any publicly accessible cluster, you should ensure all users have a password set.

**************
 Version 0.28
**************

Version 0.28.1
==============

**Release Date:** February 20, 2024

**Improvements**

-  The Google Cloud Storage client will now retry following the default policy on
   ``TooManyRequests`` rate limit errors.

**Bug Fixes**

-  Since 0.26.2, it was possible to cause Determined trials and commands to hang after the main
   process exited but before the container exited, by starting a non-terminating subprocess from
   your training script or command that kept an open ``stdout`` or ``stderr`` file descriptor. Now,
   logs from subprocesses of your main process are ignored after your main process has exited.

-  TensorBoard: Fix a bug that would allow users to view TensorBoards even if they did not have
   permission to view the corresponding workspaces.

Version 0.28.0
==============

**Release Date:** February 06, 2024

**Breaking Changes**

-  Authentication: In the enterprise edition, in the master configuration, the
   ``oidc.groups_claim_name`` setting that is used to set the string value of the authenticator's
   claim name for groups has been changed to ``oidc.groups_attribute_name``. Similarly, the
   ``oidc.display_name_claim_name`` setting that is used to set the user's display name in
   Determined has been changed to ``oidc.display_name_attribute_name``.

**New Features**

-  Experiments: Add ``resources.is_single_node`` option, which forces trials to be scheduled within
   single containers rather than across multiple nodes or pods. If the requested ``slots_per_trial``
   count is impossible to fulfill in the cluster, the experiment submission will be rejected.

**Improvements**

-  Notebooks, Shells, and Commands: On static agent-based clusters (not using dynamic cloud
   provisioning), when a ``slots`` request for a notebook, shell, or command cannot be fulfilled,
   it'll be rejected.

-  API: The checkpoint download endpoint will now allow the use of ``application/x-tar`` as an
   accepted content type in the request. It will provide a response in the form of an uncompressed
   tar file, complete with content-length information included in the headers.

**Deprecated Features**

-  API: The experiment API object in a future version will have its ``config`` field removed to
   improve performance of the system.The response of ``/api/v1/experiments/{experiment_id}`` now
   contains a new ``config`` field that can be used as a replacement. If you are not calling the
   APIs manually, there will be no impact to you.

**************
 Version 0.27
**************

Version 0.27.1
==============

**Release Date:** January 24, 2024

**New Features**

-  CLI: Add new ``--db-snapshot`` flag for the ``det deploy aws up`` subcommand that allows starting
   RDS DB instances with a pre-existing snapshot. This flag is currently only usable with the
   ``simple-rds`` deployment type.

**Improvements**

-  Notebooks: The Jupyter notebook file browser (``ContentManager``) will no longer be locked down
   to ``work_dir``, and it'll have the entire ``/`` filesystem visible. ``work_dir`` will stay the
   default starting directory.

-  Helm: Add support for downloading checkpoints when using ``shared_fs``. Add a ``mountToServer``
   value under ``checkpointStorage``. By default, this parameter is set to ``false``, preserving the
   current behavior. However, when it's set to ``true`` and the storage type is ``shared_fs``, the
   shared directory will be mounted on the server, allowing ``checkpoint.download()`` to work with
   ``shared_fs`` on Determined starting from version ``0.27.0`` and later.

Version 0.27.0
==============

**Release Date:** January 09, 2024

**Breaking Changes**

-  Experiments: Allow empty model definitions when creating experiments.

-  CLI: Optional flags must come before or after positional arguments when creating experiments;
   orderings such as ``det e create const.yaml -f .`` are no longer supported. Instead, you should
   use ``det e create -f const.yaml .`` or ``det e create const.yaml . -f``.

**Improvements**

-  Allow checkpoint downloads through the server for ``checkpoint_storage`` types ``shared_fs`` and
   ``directory``.

**************
 Version 0.26
**************

Version 0.26.7
==============

**Release Date:** December 18, 2023

**Breaking Changes**

-  CLI: Remove the ``--dry-run`` option for ``det deploy aws``. The option had no effect because AWS
   CloudFormation does not provide a way to preview staged changes.

**New Features**

-  CLI: Modify ``det user ls`` to show only active users. Add a new flag ``--all`` to show all
   users.

**New Features**

-  Authentication: *(Enterprise edition only)* SAML users can be auto-provisioned upon their first
   login. To configure, set the ``saml.auto_provision_users`` option to True. If SCIM is enabled as
   well, ``auto_provision_users`` must be False.

-  Authentication: *(Enterprise edition only)* In the enterprise edition, add synchronization of
   SAML user group memberships with existing groups and SAML user display name with the Determined
   user display name. Configure by setting ``saml.groups_attribute_name`` to the string value of the
   authenticator's attribute name for groups and ``saml.display_name_attribute_name`` with the
   authenticator's attribute name for display name.

**Improvement**

-  Security: *(Enterprise edition only)* In the enterprise edition, expand the SAML user group
   memberships feature to provision groups upon each login. This can be done by setting
   ``saml.groups_attribute_name`` to the string value of the authenticator's attribute name for
   groups. Prior releases only matched group memberships between the authenticator and local
   Determined user groups, meaning that, if not found, local groups would not be created.

-  Security: *(Enterprise edition only)* In the enterprise edition, expand the OIDC user group
   memberships feature to provision groups upon each login. This can be done by setting
   ``oidc.groups_claim_name`` to the string value of the authenticator's claim name for groups.
   Prior releases only matched group memberships between the authenticator and local Determined user
   groups, meaning that, if not found, local groups would not be created.

**Bug Fixes**

-  Master: Fix an issue where master was unable to download checkpoints from S3 buckets in the
   ``us-east-1`` region.

Version 0.26.6
==============

**Release Date:** December 07, 2023

Version 0.26.6 is a re-release of 0.26.5, which encountered some technical difficulties. The
contents of 0.26.6 are the same as 0.26.5. See release notes for 0.26.5 below.

Version 0.26.5
==============

**Release Date:** December 07, 2023

**Bug Fixes**

-  Fix an issue where ``log_policies`` would be compared against the trial log printing experiment
   config, which could often cause patterns like ``(.*) match (.*)`` to incorrectly always match.

-  Fix an issue where the ``determined.launch.wrap_rank`` module, often used by custom launch
   layers, was improperly buffering multiple lines separated by a carriage return, such as logs
   emitted from the popular TQDM library. TQDM logs will pass now through without undue buffering.

**New Features**

-  Authentication: *(Enterprise edition only)* Users can now provide a Pachyderm address in the
   master config under ``integrations.pachyderm.address``. This address will be added as an
   environment variable called ``PACHD_ADDRESS`` in task containers. The OIDC raw ID token will also
   be available as an environment variable called ``DEX_TOKEN`` in task containers.

-  Authentication: *(Enterprise edition only)* Add synchronization of OIDC user group memberships
   with existing groups. Configure by setting ``oidc.groups_claim_name`` in the master config to the
   string value of the authenticator's claim name for groups.

Version 0.26.4
==============

**Release Date:** November 17, 2023

**Breaking Changes**

-  CLI: The CLI command to patch the master log config has been changed from ``det master config
   --log --level <log_level> --color <on/off>`` to ``det master config set --log.level=<log_level>
   --log.color=<on/off>``.

**New Features**

-  Authentication: OIDC users can be auto-provisioned upon their first login. To configure, set the
   ``oidc.auto_provision_users`` option to True. If SCIM is enabled as well,
   ``auto_provision_users`` must be False.

-  Experiments: Add a ``log_policies`` configuration option to define actions when a trial's log
   matches specified patterns.

   -  The ``exclude_node`` action prevents a failed trial's restart attempts (due to its
      ``max_restarts`` policy) from being scheduled on nodes with matching error logs. This is
      useful for bypassing nodes with hardware issues like uncorrectable GPU ECC errors.

   -  The ``cancel_retries`` action prevents a trial from restarting if a trial reports a log that
      matches the pattern, even if it has remaining ``max_restarts``. This avoids using resources
      for retrying a trial that encounters certain failures that won't be fixed by retrying the
      trial, such as CUDA memory issues. For details, visit :ref:`experiment-config-reference` and
      :ref:`master-config-reference`.

   This option is also configurable at the cluster or resource pool level via task container
   defaults.

-  CLI: Add a new CLI command ``det e delete-tb-files [Experiment ID]`` to delete local TensorBoard
   files associated with a given experiment.

**Improvements**

-  Update default environment images to Python 3.9 from Python 3.8.

**Bug Fixes**

-  Users: Fix an issue where if a user's remote status was edited through ``det user edit <username>
   --remote=true``, that user could still log in using their username and password; they should only
   be able to log in through IdP integrations.

Version 0.26.3
==============

**Release Date:** November 03, 2023

**New Features**

-  CLI: Add a new CLI command ``det user edit <target_user> [--display-name] [--remote] [--active]
   [--admin] [--username]`` that allows the user to edit multiple fields for the target user. Old
   methods for editing users will still be available, but are now deprecated.

-  Add new ``directory`` checkpoint storage type, which allows for storing checkpoint and
   TensorBoard data at a specified path inside the task containers. Users are responsible for
   mounting a persistent storage at this path, e.g., a shared PVC using ``pod_spec`` configuration
   in Kubernetes-based setups.

**Deprecated Features**

-  API: Support for mixed precision in ``PyTorchTrial`` using NVIDIA's Apex library is deprecated
   and will be removed in a future version of Determined. Users should transition to Torch Automatic
   Mixed Precision (``torch.cuda.amp``). For examples, refer to the `examples
   <https://github.com/determined-ai/determined/tree/0.26.1/harness/tests/experiment/fixtures/pytorch_amp>`_.

-  Images: Environment images will no longer include the Apex package in a future version of
   Determined. If needed, users can install it from the official repository.

Version 0.26.2
==============

**Release Date:** October 25, 2023

Notice: The ``ruamel.yaml`` library's 0.18.0 release includes breaking changes that affect earlier
versions of Determined. The failure behavior is that commands that emit YAML, such as ``det
experiment config``, will emit nothing to ``stdout`` or ``stderr`` but instead silently exit 1 due
to the new version of ``ruamel.yaml``. This release of Determined has included a
``ruamel.yaml<0.18.0`` requirement, but older versions of Determined will also be affected, so users
of older versions of Determined may have to manually downgrade ``ruamel.yaml`` if they observe this
behavior.

**New Features**

-  Python SDK: Add various new features and enhancements. A few highlights are listed below.

   -  Add support for downloading a zipped archive of experiment code
      (:meth:`Experiment.download_code <determined.experimental.client.Experiment.download_code>`).

   -  Add support for :class:`~determined.experimental.client.Project` and
      :class:`~determined.experimental.client.Workspace` as SDK objects.

   -  Surface more attributes to resource classes, including ``hparams`` and ``summary_metrics`` for
      :class:`~determined.experimental.client.Trial`.

   -  Add support for fetching and filtering multiple experiments with
      :meth:`client.list_experiments <determined.experimental.client.list_experiments>`.

   -  Add support for filtering trial logs by timestamp and a query string using
      :meth:`Trial.iter_logs <determined.experimental.client.Trial.iter_logs>`.

   -  All resource objects now have a ``.reload()`` method that refreshes the resource's attributes
      from the server. Previously, attributes were most easily refreshed by creating an entirely new
      object.

-  Python SDK: All ``GET`` API calls now retry the request up to 5 times on failure.

**Deprecated Features**

-  Python SDK: Several methods have been renamed for better API standardization.

   -  Methods returning a ``List`` and ``Iterator`` now have names starting with ``list_*`` and
      ``iter_*``, respectively.

   -  :class:`~determined.experimental.client.TrialReference` and
      :class:`~determined.experimental.client.ExperimentReference` are now
      :class:`~determined.experimental.client.Trial` and
      :class:`~determined.experimental.client.Experiment`.

-  Python SDK: Consolidate various ways of fetching checkpoints.

   -  :meth:`Experiment.top_checkpoint <determined.experimental.client.Experiment.top_checkpoint>`
      and :meth:`Experiment.top_n_checkpoints
      <determined.experimental.client.Experiment.top_n_checkpoints>` are deprecated in favor of
      :meth:`Experiment.list_checkpoints
      <determined.experimental.client.Experiment.list_checkpoints>`.

   -  :meth:`Trial.get_checkpoints <determined.experimental.client.Trial.get_checkpoints>`,
      :meth:`Trial.top_checkpoint <determined.experimental.client.Trial.top_checkpoint>`, and
      :meth:`Trial.select_checkpoint <determined.experimental.client.Trial.select_checkpoint>` are
      deprecated in favor of :meth:`Trial.list_checkpoints
      <determined.experimental.client.Trial.list_checkpoints>`.

-  Python SDK: Deprecate resource ordering enum classes (``CheckpointOrderBy``,
   ``ExperimentOrderBy``, ``TrialOrderBy``, ``ModelOrderBy``) in favor of a shared
   :class:`~determined.experimental.client.OrderBy`.

**Bug Fixes**

-  Core API: On context closure, properly save all TensorBoard files not related to metrics
   reporting, particularly the native profiler traces.
-  Core API v2: Fix an issue where TensorBoard files were not saved for managed experiments.

Version 0.26.1
==============

**Release Date:** October 12, 2023

**New Features**

-  Experiments: Add an experiment continue feature to the CLI (``det e continue <experiment-id>``),
   which allows for resuming or recovering training for an experiment whether it previously
   succeeded or failed. This is limited to single-searcher experiments and using it may prevent the
   user from replicating the continued experiment's results.

**Improvements**

-  Logging: Some API logs would previously only go to the standard output of the running master but
   now will also appear in the output of ``det master logs``.

-  Kubernetes: Increase the file context limit for notebooks, commands, TensorBoards, and shells
   from approximately 1MB to roughly 95MB, the same limit as the agent resource manager.

-  CLI: ``det notebook|shell|tensorboard open <id>`` will now wait for the item to be ready instead
   of giving an error if it is not ready.

-  Detached mode: Add support for S3 and GCS cloud storage for TensorBoard files.

-  Kubernetes: On Kubernetes, ``max_slots_per_pod`` can now be configured at a resource pool level
   through the master config option
   ``resource_pools.task_container_defaults.kubernetes.max_slots_per_pod``.

**Bug Fixes**

-  TensorBoard: Fix an issue where TensorBoard files for an experiment were not getting deleted when
   the experiment was deleted.

-  Kubernetes: Fix an issue where custom node affinities on tasks were being ignored.

   On Kubernetes, upgrading from a version before this feature to a version after this feature can
   cause queued allocations with a custom node affinity to be killed. Users can pause queued
   experiments to avoid this.

**Known Issue**

-  When using custom metric groups, the ``Learning Curve`` view in the experiment's visualization
   tab does not render.

Version 0.26.0
==============

**Release Date:** September 25, 2023

**Breaking Changes**

-  Kubernetes: Remove the ``agent_reattach_enabled`` config option. Agent reattach is now always
   enabled.
-  Agent: Take the default value for the ``--visible-gpus`` option from the ``CUDA_VISIBLE_DEVICES``
   or ``ROCR_VISIBLE_DEVICES`` environment variables, if defined.

**New Features**

-  SDK: Add the ability to keep track of what experiments use a particular checkpoint or model
   version for inference.

-  SDK: Add :meth:`Checkpoint.get_metrics <determined.experimental.client.Checkpoint.get_metrics>`
   and :meth:`ModelVersion.get_metrics <determined.experimental.model.ModelVersion.get_metrics>`
   methods.

-  Kubernetes: Support enabling and disabling agents to prevent Determined from scheduling jobs on
   specific nodes.

   Upgrading from a version before this feature to a version after this feature only on Kubernetes
   will cause queued allocations to be killed on upgrade. Users can pause queued experiments to
   avoid this.

**Improvements**

-  Enable reporting and display of metrics with floating-point epoch values.

-  API: Allow the reporting of duplicate metrics across multiple ``report_metrics`` calls with the
   same ``steps_completed``, provided they have identical values.

-  SDK: :func:`~determined.experimental.client.stream_trials_training_metrics` and
   :func:`~determined.experimental.client.stream_trials_validation_metrics` are now deprecated.
   Please use :func:`~determined.experimental.client.stream_trials_metrics` instead. The
   corresponding methods of :class:`~determined.experimental.client.Determined` and
   :class:`~determined.experimental.client.TrialReference` have also been updated similarly.

**Bug Fixes**

-  Checkpoints: Fix an issue where in certain situations duplicate checkpoints with the same UUID
   would be returned by the WebUI and the CLI.
-  Models: Fix a bug where ``det model describe`` and other methods in the CLI and SDK that act on a
   single model would error if two models had similar names.
-  Workspaces: Fix an issue where notebooks, TensorBoards, shells, and commands would not inherit
   agent user group and agent user information from their workspace.

**************
 Version 0.25
**************

Version 0.25.1
==============

**Release Date:** September 11, 2023

**Breaking Changes**

-  Fluent Bit is no longer used for log shipping and configs associated with Fluent Bit are now no
   longer in use. Fluent Bit has been replaced with an internal log shipper (the same one that is
   used for Slurm).

**Bug Fixes**

-  Reduce the time before seeing the first metrics of a new experiment.

Version 0.25.0
==============

**Release Date:** August 29, 2023

**Breaking Changes**

-  Remove ``EstimatorTrial``, which has been deprecated since Determined version 0.22.0 (May 2023).

**Bug Fixes**

-  Trials: Fix an issue where trial logs could fail for trials created prior to Determined version
   0.17.0.
-  CLI: Fix an issue where template association with workspaces, when listed, was missing. This
   would prevent templates from being listed for some users and templates on RBAC-enabled clusters.

**************
 Version 0.24
**************

Version 0.24.0
==============

**Release Date:** August 18, 2023

**Breaking Changes**

-  API: Remove ``LightningAdapter``, which was deprecated in 0.23.1 (June 2023). We recommend that
   PyTorch Lightning users migrate to the :ref:`Core API <core-getting-started>`.

**New Features**

-  Environments: Add experimental PyTorch 2.0 images containing PyTorch 2.0.1, Python 3.10.12, and
   (for the GPU image) CUDA 11.8.

**Bug Fixes**

-  Users: Fix an issue that caused the CLI command ``det user list`` to always show "false" in the
   "remote" column.

**************
 Version 0.23
**************

Version 0.23.4
==============

**Release Date:** July 31, 2023

**Breaking Changes**

-  API: The ``/api/v1/users/setting`` endpoint no longer accepts ``storagePath`` and now accepts a
   ``settings`` array instead of a single ``setting``.

**New Features**

-  Allow non-intersecting dictionaries of metrics to be merged on the same ``total_batches``. This
   update was rejected before.

-  API: Add a new patch API endpoint ``/api/v1/master/config`` that allows the user to make changes
   to the master config while the cluster is running. Currently, only changing the log config is
   supported.

-  CLI: Add a new CLI command ``det master config --log --level <log_level> --color <on/off>`` that
   allows the user to change the log level and color settings of the master config while the cluster
   is still running. ``det master config`` can still be used to get the master config.

-  Cluster: Allow binding resource pools to specific workspaces. Bound resource pools can only be
   used by the workspaces they are bound to. Each workspace can also now have a default compute
   resource pool and a default auxiliary resource pool configured.

-  Kubernetes: Users may now populate all ``securityContext`` fields within the pod spec of the
   ``determined-container`` container except for ``RunAsUser`` and ``RunAsGroup``. For those fields,
   use ``det user link-with-agent-user`` instead.

-  WebUI: The experiment list page now has the following new capabilities:

   -  Select metrics and hyperparameters as columns.
   -  Filter the list on any available column.
   -  Specify complex filters.
   -  Sort the list on any available column.
   -  Display total number of experiments matching the filter.
   -  Compare metrics, hyperparameters, and trial details across experiments.
   -  Toggle between pagination and infinite scroll.
   -  Select preferred table density.

**Improvements**

-  WebUI: Improve performance and stability.

Version 0.23.3
==============

**Release Date:** July 18, 2023

**Breaking Changes**

-  API: Remove the ``/config`` endpoint, replaced by ``/api/v1/master/config``.

**Improvements**

-  Notebooks: Upgrade the connection between the master and notebook tasks to use HTTPS for enhanced
   security.

**Deprecated Features**

-  API: Remove the ``SummarizeTrial`` endpoint favor of ``CompareTrials``; ``CompareTrials`` sends a
   similar request with the ``trial_id`` parameter replaced by the ``trial_ids`` array.
-  API: Remove the ``scale`` from the ``CompareTrialsRequest`` endpoint; this was used only for LTTB
   downsampling, which has since been replaced.

Version 0.23.2
==============

**Release Date:** July 05, 2023

**New Features**

-  CLI: ``det deploy gcp up`` now uses a default Google Cloud Storage bucket
   ``$PROJECT-ID-determined-deploy`` to store the Terraform state unless a local Terraform state
   file is present or a different Cloud Storage bucket is specified.

-  CLI: A new list function ``det deploy gcp list --project-id <project_id>`` was added that lists
   all clusters under the default Cloud Storage bucket in the given project. Clusters from a
   particular Cloud Storage bucket can also be listed using ``det deploy gcp list --project-id
   <project_id> --tf-state-gcs-bucket-name <tf_state_gcs_bucket_name>``.

-  CLI: A new delete subcommand ``det deploy gcp down --cluster-id <cluster_id> --project-id
   <project_id>`` was added that deletes a particular cluster from the project. ``det deploy gcp
   down`` can still be used to delete clusters with local Terraform state files.

Version 0.23.1
==============

**Release Date:** June 21, 2023

**Improvements**

-  Errors: Errors that return 404 or 'Not Found' codes now have standardized messaging using the
   format "<task/trial/workspace etc.> <ID> not found". In addition, if RBAC is enabled, the error
   message includes a suffix to remind users to check their permissions. This is because with RBAC
   enabled, permission denied errors and not found errors both return a 'Not Found' response.

**Deprecated Features**

-  ``LightningAdapter`` is deprecated and will be removed in a future version. We recommend that
   PyTorch Lightning users migrate to the :ref:`Core API <core-getting-started>`.

**Bug Fixes**

-  Users: Resolved an issue that was causing an error when attempting to create a new user with a
   username that was previously used by a renamed user.

Version 0.23.0
==============

**Release Date:** June 05, 2023

**Breaking Changes**

-  Remove HDFS checkpoint storage support, which has been deprecated since 0.21.1 (April 2023).

-  Kubernetes: When a pod spec is specified in both ``task_container_defaults`` and the
   experiment/job configuration, the pod spec is merged according to `strategic merge patch
   <https://kubernetes.io/docs/tasks/manage-kubernetes-objects/update-api-object-kubectl-patch/#use-a-strategic-merge-patch-to-update-a-deployment>`__.
   The previous behavior was using only the experiment/job configuration if supplied.

-  CLI: The ``det notebook|tensorboard start`` commands no longer block for the whole life cycle of
   the notebook or TensorBoard process. They will also not stream related event logs. Users should
   use the existing ``det notebook|tensorboard|task logs`` commands to stream logs from the process.

-  Python SDK: Remove the packages ``determined-cli``, ``determined-common``, and
   ``determined-deploy``, which were deprecated in 0.15.0 (April 2021). The submodules
   ``determined.cli``, ``determined.common``, and ``determined.deploy`` of the ``determined``
   package should be used instead.

**New Features**

-  Experiment: Custom hyperparameter searchers can include extra directories to pass into the
   ``client.create_experiment`` context.

-  Checkpoints: Add support for deleting a subset of files from checkpoints.

   The SDK method :meth:`determined.experimental.client.Checkpoint.remove_files` has been added to
   delete files matching a list of globs provided. The CLI command ``det checkpoint rm uuid1,uuuid2
   --glob 'deleteDir1/**' --glob deleteDir2`` provides access to this method.

-  AWS and GCP: Add ``launch_error_timeout`` and ``launch_error_retries`` provider configuration
   options.

   -  ``launch_error_timeout``: Duration for which a provisioning error is valid. Tasks that are
      unschedulable in the existing cluster may be canceled. After the timeout period, the error
      state is reset. Defaults to ``0s``.

   -  ``launch_error_retries``: Number of retries to allow before registering a provider
      provisioning error. Defaults to ``0``.

-  DeepSpeed experiments can now be wrapped with the ``determined.pytorch.dsat`` module to
   automatically tune their distributed training hyperparameters.

-  API: ``GetExperiments(archived=False)`` no longer lists experiments from archived projects or
   workspaces. This change affects both the WebUI and the CLI. Unarchived projects and workspaces
   are not affected.

**Improvements**

-  CLI: ``det user list`` will not display the Admin column when RBAC is enabled.
-  Checkpoints: In checkpoint-related views and APIs, the previously hidden file ``metadata.json``
   is now visible.

**************
 Version 0.22
**************

Version 0.22.2
==============

**Release Date:** May 24, 2023

**Improvements**

-  Cluster: Slurm/PBS requires HPC Launcher 3.2.9.

   -  The HPC Launcher includes new support to enable improved scalablity. When used with Slurm or
      PBS, the launcher must be version 3.2.9 or greater.

-  Bind mounts for notebooks (and other commands) can be configured with ``--config``. For example
   usage, see the section for ``--config`` in ``det command run --help``.

-  Trials: Reporting a training or validation metric with the epoch set to a non-numeric value will
   now return an error.

**Deprecated Features**

-  CLI: ``det template set <name> <config>`` has been deprecated.

**Removed Features**

-  API: Legacy APIs for trial details and trial metrics, which were deprecated in 0.19.2, have now
   been removed.
-  API: Legacy APIs for experiment creation and updates, which were deprecated in 0.19.10, have now
   been removed.

**Bug Fixes**

-  CLI: ``det e list`` and ``det e list -a`` behaviors were erroneously switched.

   -  Earlier, ``det e list`` was showing both archived and unarchived experiments, and ``det e list
      -a`` was showing only unarchived experiments. This has now been fixed --- ``det e list`` will
      show only unarchived experiments and ``det e list -a`` will show both archived and unarchived
      experiments.

Version 0.22.1
==============

**Release Date:** May 17, 2023

**Bug Fixes**

-  Fix a critical regression in 0.22.0 that could lead to database deadlocks and incorrect
   experiment progress info when restarting trials after failure. Specifically, this problem may
   occur when the ``max_restarts`` experiment configuration option is set to a value greater than
   zero (default: 5). We advise all users running 0.22.0 to upgrade as soon as possible.

Version 0.22.0
==============

**Release Date:** May 05, 2023

**Breaking Change**

-  The previous template CRUD endpoints have been removed from the ``/templates/*`` location. Please
   use the APIs found at ``/api/v1/templates/*``.

-  Experiment: Optimizer must be an instance of ``tensorflow.keras.optimizers.legacy.Optimizer``
   starting from Keras 2.11.

   -  Experiments now use images with TensorFlow 2.11 by default. TensorFlow users who are not
      explicitly configuring their training images will need to adapt their model code to reflect
      these changes. Users will likely need to use Keras optimizers located in
      ``tensorflow.keras.optimizers.legacy``. Depending on the sophistication of users' model code,
      there may be other breaking changes. Determined is not responsible for these breakages. See
      the `TensorFlow release notes
      <https://github.com/tensorflow/tensorflow/releases/tag/v2.11.0>`_ for more details.

   -  PyTorch users and users who specify custom images should not be affected.

**Deprecated Features**

-  Legacy TensorFlow 1 + PyTorch 1.7 + CUDA 10.2 support is deprecated and will be removed in a
   future version. The final TensorFlow 1.15.5 patch was released in January 2021, and no further
   security patches are planned. Consequently, we recommend users migrate to modern versions of
   TensorFlow 2 and PyTorch. Our default environment images currently ship with
   ``tensorflow==2.11.1`` and ``torch==1.12.0``.

-  ``EstimatorTrial`` is deprecated and will be removed in a future version. TensorFlow has advised
   Estimator users to switch to Keras since TensorFlow 2.0 was released. Consequently, we recommend
   users of EstimatorTrial switch to the :class:`~determined.keras.TFKerasTrial` class.

-  Master config option ``logging.additional_fluent_outputs`` is deprecated and will be removed in a
   future version. We do not plan to offer a replacement at this time. If you are interested in
   additional logging integrations, please contact us.

**Improvement**

-  HP Search: Trials are persisted as soon as they are requested by the searcher, instead of after
   they are first scheduled.

-  Trials: Metric storage has been optimized for reading summaries of metrics reported during a
   trial.

   Extended downtime may result when upgrading from a previous version to this version or a later
   version. This will occur when your cluster contains a large number of trials and training steps
   reported. For example, a database with 10,000 trials with 125 million training metrics on a small
   instance may experience 6 or more hours of downtime during the upgrade.

   (Optional) To minimize downtime, users with large databases can choose to manually run `this SQL
   file
   <https://github.com/determined-ai/determined/blob/main/master/static/migrations/20230503144448_add-summary-metrics.tx.up.sql>`__
   against their cluster's database while it is still running before upgrading to a new version.
   This is an optional step and is only recommended for significantly large databases.

**************
 Version 0.21
**************

Version 0.21.2
==============

**Release Date:** April 28, 2023

**New Features**

-  Add the ``launch_error`` configuration option to the master config, which specifies whether to
   refuse experiments or tasks if they request more slots than the cluster has. See
   :ref:`master-config-reference` for more information.

**Improvements**

-  CLI: Add ``det (experiment|trial|task) logs --json`` option, allowing users to get JSON-formatted
   logs for experiments, trials, and tasks.

-  Cluster: HPC Launcher 3.2.7 migrates the ``resource_manager.job_storage_root`` to a more
   efficient format. This happens automatically, but once migrated you cannot downgrade to an older
   version of the HPC launcher.

-  Cluster: The ``manage-singularity-cache`` script has added the ``--docker-login`` option to
   enable access to private Docker images.

**Removed Features**

-  The "hyperparameter importance" feature and associated API endpoints have been removed.

**Bug Fixes**

-  Tasks: Fix an issue where task proxies were not recovered when running on Slurm.
-  Tasks: Fix an issue where ``det task list`` would sometimes return an incorrect 404 error.

Version 0.21.1
==============

**Release Date:** April 11, 2023

**Breaking Change**

-  Remove old master logs ``/logs`` endpoint. Users should use ``/api/v1/master/logs`` instead.

**Bug Fixes**

-  Fix an issue introduced in 0.19.9 where ``task_container_defaults`` for the default resource
   pools were not respected for experiments and tasks unless they specified the resource pool name
   explicitly.

-  Checkpoints: Fix an issue where checkpoint insertion on a cluster with a lot of checkpoints and
   reported metrics could take a long time.

-  Kubernetes: Fix a crash affecting zero-slot workloads when ``resources.limits`` and
   ``resources.requests`` overrides were explicitly specified in the pod spec.

**Deprecated Features**

-  HDFS checkpoint storage support has been deprecated and will be removed in a future version.
   Please contact Determined if you still need it, or else migrate to a different storage backend.

**Improvement**

-  Cluster: Add HPC Launcher support for JVM resource configuration.

   -  The master configuration option ``resource_manager.launcher_jvm_args`` can be used to override
      the default HPC Launcher JVM heap configuration. This support requires HPC Launcher version
      3.2.6 or greater.

**New Features**

-  Python SDK: Add methods for efficient export of training and validation metrics to the Python
   SDK. The methods are listed below.

   -  :meth:`~determined.experimental.client.stream_trials_training_metrics`
   -  :meth:`~determined.experimental.client.stream_trials_validation_metrics`
   -  :meth:`~determined.experimental.client.Trial.stream_training_metrics`
   -  :meth:`~determined.experimental.client.Trial.stream_validation_metrics`

**Removed Features**

-  The separate ``det-deploy`` executable was deprecated in 0.15.0 (April 2021) and is now removed.
   Use the ``det deploy`` subcommand instead.

Version 0.21.0
==============

**Release Date:** March 27, 2023

**Breaking Changes**

-  Cluster: K80 GPUs are no longer supported.

-  API: Remove all old PATCH endpoints under ``/agents*``, including the APIs for enabling and
   disabling slots. Users should use the new APIs under ``/api/v1/agents``.

-  API: The ``on_validation_step_start`` and ``on_validation_step_end`` callbacks on
   ``PyTorchTrial`` and ``DeepSpeedTrial`` were deprecated in 0.12.12 (Jul 2020) and have been
   removed. Please use ``on_validation_start`` and ``on_validation_end`` instead.

-  Trial API: ``records_per_epoch`` has been dropped from PyTorch code paths. We were previously
   using this value internally to estimate epoch lengths. We are now using the chief worker's epoch
   length as the epoch length.

-  API: ``average_training_metrics`` is no longer configurable. This value previously defaulted to
   false and was dropped to simplify the training API. We always average training metrics now.

-  API: The unused ``latest_training`` field has been removed from the ``GetTrial`` and
   ``GetExperimentTrials`` APIs due to slow performance.

**Bug Fixes**

-  CLI: Fix an issue where ``det user change-password`` would return an authentication error when
   trying to change the current user's password.

**Improvements**

-  CLI: Command-line deployments will now default to provisioning NVIDIA T4 GPU instances instead of
   K80 instances. This change is intended to improve the performance/cost and driver support of the
   default deployment.

-  Kubernetes: Ease permission requirements in Kubernetes so master no longer requires access to all
   Kubernetes namespaces. This only affects custom modified Helm chart configurations.

-  Checkpoints: Improve performance of checkpoint insertion and deletion.

**New Feature**

-  API: Deprecate ``TorchWriter`` and add a PyTorch ``SummaryWriter`` object to
   ``PyTorchTrialContext`` and ``DeepSpeedTrialContext`` that we manage on behalf of users. See
   :func:`~determined.pytorch.PyTorchTrialContext.get_tensorboard_writer` for details.

-  API: Introduce :class:`~determined.pytorch.Trainer`, a high-level training API for
   ``PyTorchTrial`` that allows for Python-side training loop customizations and includes support
   for off-cluster local training.

**Removed Features**

-  The following methods of :class:`~determined.experimental.client.Checkpoint`,
   :class:`~determined.experimental.client.Model`, and
   :class:`~determined.experimental.client.ModelVersion` were deprecated in 0.17.9 (Feb 2022) and
   are now removed:

   -  ``Checkpoint.load()``
   -  ``Checkpoint.load_from_path()``
   -  ``Checkpoint.parse_metadata()``
   -  ``Checkpoint.get_type()``
   -  ``Checkpoint.from_json()``
   -  ``Model.from_json()``
   -  ``ModelVersion.from_json()``

**************
 Version 0.20
**************

Version 0.20.1
==============

**Release Date:** March 15, 2023

**Breaking Changes**

-  Database: Several unused columns have been dropped from the ``raw_steps``, ``raw_validations``,
   and ``raw_checkpoints`` database tables. The database migration will involve a sequential scan
   for these tables, and it may take a significant amount of time, depending on the database size
   and performance.

**New Features**

-  Tasks and experiments can now expose arbitrary ports that you can tunnel to using the CLI. To
   learn more about how to expose custom ports or see an example, check out :ref:`proxy-ports` or
   visit ``examples/features/ports``.

-  Container Images: Add maintained images for PyTorch-only environments. The current environment
   images contain both PyTorch and TensorFlow, resulting in large image sizes. The new images are
   appropriate for users who do not require TensorFlow but may still require TensorBoard.

**Removed Features**

-  API: Remove internal ``ExpCompareMetricNames`` and ``ExpCompareTrialsSample`` endpoints, which
   have been unused and deprecated since 0.19.5.

**Known Issue**

-  For multi-trial experiments, training metrics do not start appearing unless there has been at
   least one validation.

Version 0.20.0
==============

**Release Date:** February 28, 2023

**Breaking Changes**

-  Cluster: The ``resources.agent_label`` task option and ``label`` agent config option are no
   longer supported and will be ignored. If you are not explicitly using these options, or only use
   a single empty or non-empty label value per resource pool, no changes are necessary. Otherwise,
   cluster admins should create a resource pool for each existing ``resource_pool``/``agent_label``
   combination and reconfigure agents to use these new pools. Cluster users should update their
   tasks to use the new resource pool names.

**Bug Fixes**

-  Model Registry: Fix an issue where a model with versions from multiple workspaces could have its
   versions modified by a user with edit access to only a single one of those workspaces.
-  WebUI: Patch an issue where logging out would not properly redirect to the login page.
-  WebUI: Fix a bug where the cluster's job queue page could crash in certain cases.

**Improvements**

-  Agents: The master configuration ``agent_reattach_enabled`` is always enabled and agents will now
   always reattach containers on restart.

-  Kubernetes: The cluster information page now takes resource quotas into account if there are any
   on relevant namespaces.

-  RBAC: Model registry models and commands that are inaccessible to the user will appear as
   uneditable. Previously, users could attempt the action and would encounter a permission denied
   error.

-  CLI: When listing TensorBoards, show ``workspaceName`` instead of ``workspaceId`` for better
   readability and prevent N/A values from appearing.

**New Features**

-  RBAC: Following on the initial RBAC support added in 0.19.7, the enterprise edition of Determined
   (`HPE Machine Learning Development Environment
   <https://www.hpe.com/us/en/hpe-machine-learning-development-environment.html>`_) has added
   support for role-based access control (RBAC) over new entities:

   -  Notebooks, TensorBoards, shells, and commands are now housed under workspaces. Access to these
      tasks can now be restricted by role.
   -  Model Registry: Models are now associated with workspaces. Models can be moved between
      workspaces and access to them can be restricted by role.

   These changes allow for more granular control over who can access what resources. See :ref:`rbac`
   for more information.

**************
 Version 0.19
**************

Version 0.19.11
===============

**Release Date:** February 17, 2023

**Bug Fixes**

-  Kubernetes: Fix an issue where environment variables with an equals character in the value, such
   as ``func=f(x)=x``, were processed incorrectly in Kubernetes.
-  Agent: Fix a bug where if agent reattach was enabled and the master was down while an active
   task's Docker container failed, the task could get stuck in an unkillable running state.
-  ``det deploy aws``: Update CloudFormation permissions to allow checkpoint downloads through
   master.
-  Tasks: Fix a bug where in rare cases tasks could take an extra 30 seconds to complete.

**Improvements**

-  Container Images: Publish multi-arch master and agent container image manifests with AMD64,
   ARM64, and PPC64 architectures.

-  Experiments: If an experiment with no checkpoints is deleted, a checkpoint GC task will no longer
   be launched. Launching a checkpoint GC task could prevent experiments with certain incorrect
   configuration from being deleted.

-  Cluster: Capability added for checkpoint downloads from Google Cloud Storage via a master
   instance.

-  Installation: ``.deb`` and ``.rpm`` Linux packages will now install master and agent binaries
   into ``/usr/bin/`` instead of ``/usr/local/bin/``, to be more in line with the Filesystem
   Hierarchy Standard.

-  Kubernetes: Empty environment variables can now be specified in Kubernetes, while before they
   would throw an error.

-  Kubernetes: Zero-slot tasks on GPU clusters will not request ``nvidia.com/gpu: 0`` resources any
   more, allowing them to be scheduled on CPU-only nodes.

-  Installation: Add experimental Homebrew (macOS) package.

-  Scheduler: The scheduler can be configured to find fits for distributed jobs against agents of
   different sizes.

**New Features**

-  CLI: Add a ``--add-tag`` flag to AWS ``det deploy aws up``, which specifies tags to add to the
   underlying CloudFormation stack.

   -  New tags will not replace automatically added tags such as ``deployment-type`` or
      ``managed-by``.

   -  Any added tags that should persist across updates should be always be included when using
      ``det deploy aws up`` -- if the argument is missing, any previously added tags would be
      removed.

Version 0.19.10
===============

**Release Date:** January 20, 2023

**Breaking Changes**

-  Kubernetes: Add the ``kubernetes_namespace`` config field for resource pools, specifying a
   Kubernetes `namespace
   <https://kubernetes.io/docs/concepts/overview/working-with-objects/namespaces/>`__ that tasks
   will be launched into.

-  The name of the resource pool in Kubernetes has changed from ``"kubernetes"`` to ``"default"``.
   Forked experiments will need to have their configurations manually modified to update the
   resource pool name.

**New Features**

-  Cluster: Add support for experiment tag propagation.

   -  The enterprise edition of Determined (`HPE Machine Learning Development
      <https://www.hpe.com/us/en/hpe-machine-learning-development-environment.html>`_) now allows
      for experiment tags to be propagated as labels to the associated jobs on the HPC cluster. A
      number of labeling schemes are supported, controlled by the configuration item
      ``resource_manager.job_project_source``.

-  Cluster: Add support for launcher-provided resource pools.

   -  The enterprise edition of Determined (`HPE Machine Learning Development
      <https://www.hpe.com/us/en/hpe-machine-learning-development-environment.html>`_) now allows
      for custom resource pools to be defined that submit work to an underlying Slurm/PBS partition
      on an HPC cluster with different submission options.

-  Cluster: Determined Enterprise Edition now supports the `NVIDIA Enroot
   <https://github.com/NVIDIA/enroot>`__ container platform as an alternative to
   Apptainer/Singularity/Podman.

**Improvements**

-  Notebooks: The default idle notebook termination timeout can now be set via the
   ``notebook_timeout`` master config option.

-  Trials: Trials can now be killed when in the ``STOPPING_CANCELED`` state. Previously, if a trial
   did not implement preemption correctly and was canceled, the trial did not stop and was
   unkillable until the preemption timeout of an hour.

**Bug Fixes**

-  Fix a bug where notebooks, TensorBoards, shells, and commands restored after a master restart
   would have a submission time of when the master restarted rather than the original job submission
   time.

-  ``det deploy aws``: Fix reliability issue in ``efs`` deployment type, fix broken ``fsx``
   deployment type.

-  Job queue: Fix an issue where the CLI command ``det job list`` would ignore the argument
   ``--resource-pool``.

-  Distributed training: Fix a bug where a distributed training trial that called
   ``context.set_stop_requested`` would cause the trial to error and prevent it from completing
   successfully.

**Removed Features**

-  The data layer feature, which was deprecated in 0.18.0 (May 2022), has been removed. A migration
   guide to use the underlying yogadl library directly may be found `here
   <https://gist.github.com/rb-determined-ai/60813f1f75f75e3073dfea351a081d7e>`_. Affected users are
   encouraged to follow the migration guide before upgrading to avoid downtime.

Version 0.19.9
==============

**Release Date:** December 20, 2022

**New Features**

-  WebUI: Display total checkpoint size for experiments.

-  WebUI: Add links from forked experiments and continued trials to their parents.

-  API: Add structured fields to task log objects.

-  Cluster: Add support for launcher-provided resource pools. Determined Enterprise Edition now
   allows for custom resource pools to be defined that submit work to an underlying Slurm/PBS
   partition on an HPC cluster with different submission options.

-  Cluster: Determined Enterprise Edition now supports the `NVIDIA Enroot
   <https://github.com/NVIDIA/enroot>`__ container platform as an alternative to
   Apptainer/Singularity/Podman.

Version 0.19.8
==============

**Release Date:** December 02, 2022

**Breaking Changes**

-  API: The ``GetModelVersion``, ``PatchModelVersion``, and ``DeleteModelVersion`` APIs now take a
   sequential model version number ``model_version_num`` instead of a surrogate key
   ``model_version_id``.

**Bug Fixes**

-  Experiment: Fix an issue where experiments created before version 0.16.0 could have issues
   loading.
-  Python SDK: Fix an issue where the Model Registry call ``model.get_version(version)`` did not
   work when a specific version was passed.

**Improvements**

-  Kubernetes: If a pod exits and Determined cannot get the exit code, the code will be set to 1025
   instead of 137 to avoid confusion with potential out-of-memory issues.
-  API: Patching a user will no longer make partial updates if an error occurs.
-  Kubernetes: Specifying ``tensorboardTimeout`` in Helm will cause the specified timeout to be
   applied.
-  AWS: ``det deploy aws`` will use IMDSv2 for improved security.

**New Features**

-  Experiment: Determined Enterprise Edition now allows control of the GPU type within a Slurm GRES
   expression. If you have partitions with mixed GPU types, you may now specify the desired type
   using the ``slurm.gpu_type`` attribute of the experiment configuration.

Version 0.19.7
==============

**Release Date:** November 14, 2022

**New Features**

-  WebUI: Adds support for creating and managing webhooks to enable receiving updates regarding
   experiment state changes.

-  Checkpoint storage can now be configured at a workspace level. Experiments created in projects
   will now inherit checkpoint storage configuration from the project's workspace if set. Experiment
   configuration can override the workspace level checkpoint storage configuration.

-  Example: Textual Inversion training and generation using Stable Diffusion with Core API and
   Hugging Face's Diffusers.

-  Python SDK now supports reading logs from trials, via the new
   :meth:`~determined.experimental.client.Trial.logs` method. Additionally, the Python SDK also
   supports a new blocking call on an experiment to get the first trial created for an experiment
   via the :meth:`~determined.experimental.client.ExperimentReference.await_first_trial()` method.
   Users who have been writing automation around the ``det e create --follow-first-trial`` CLI
   command may now use the Python SDK instead, by combining ``.await_first_trial()`` and
   ``.logs()``.

-  RBAC: the enterprise edition of Determined (`HPE Machine Learning Development Environment
   <https://www.hpe.com/us/en/hpe-machine-learning-development-environment.html>`_) has added
   preliminary support for Role-Based Access Control. Administrators can now configure which users
   or user groups can administer users, create or configure workspaces, run or view experiments in
   particular workspaces, or perform other actions. See :ref:`rbac` for more information.

**Bug Fixes**

-  Master: Correctly handle pending allocations in historical resource allocation aggregation.

Version 0.19.6
==============

**Release Date:** October 28, 2022

**Breaking Changes**

-  API: Remove the legacy endpoint ``/tasks/:task_id`` due to it always incorrectly returning a
   missing parameter.

-  Experiment: Additional Slurm options formerly specified in the experiment environment section are
   now part of a new ``slurm`` section of the experiment configuration. For example, what was
   formerly written as

   .. code:: yaml

      environment:
      ...
        slurm:
          - --mem-per-cpu=10
          - --exclusive

   is now specified as

   .. code:: yaml

      environment:
      ...
      slurm:
        sbatch_args:
          - --mem-per-cpu=10
          - --exclusive

**Improvements**

-  CLI: Add the ``ls`` abbreviation for ``list`` to all applicable CLI commands.

-  CLI: Support a new ``-i``/``--include`` option in task-starting CLI commands. The context option
   (``--context``) is useful for copying a directory of files into the task container, but it may
   only be provided once, and it can be clunky if you only care about one or two files. The
   ``--include`` option also copies files into the task container, but:

   -  The directory name is preserved, so ``-i my_data/`` would result in a directory named
      ``my_data/`` appearing in the working directory of the task container.
   -  It may point to a file, so ``-i my_data.csv`` will place ``my_data.csv`` into the working
      directory.
   -  It may be specified multiple times to include multiple files and/or directories.

-  **Breaking Change:** ``det deploy aws`` by default now configures agent instances to
   automatically shut down if they lose their connection to the master. The
   ``--no-shut-down-agents-on-connection-loss`` option can be used to turn off this behavior.

**New Features**

-  Custom Searcher: users can now define their own logic to coordinate across multiple trials within
   an experiment. Examples of use cases are custom hyperparameter searching algorithms, ensembling,
   active learning, neural architecture search, reinforcement learning.

-  Cluster: The enterprise edition of `HPE Machine Learning Development Environment
   <https://www.hpe.com/us/en/hpe-machine-learning-development-environment.html>`_ can now be
   deployed on a PBS cluster. When using PBS scheduler, HPE Machine Learning Development Environment
   delegates all job scheduling and prioritization to the PBS workload manager. This integration
   enables existing PBS workloads and HPE Machine Learning Development Environment workloads to
   coexist and access all of the advanced capabilities of the PBS workload manager. You can use
   either Singularity or Podman for the container runtime.

Version 0.19.5
==============

**Release Date:** October 10, 2022

**Improvements**

-  Added the ability to set what Unix user and group tasks will run as on the agent at the workspace
   level. The setting takes precedence over users' individual user and group settings.
-  CLI: The ``det workspace edit`` command now accepts a new workspace name as an optional
   ``--name`` flag, e.g., ``det workspace edit OLD_WORKSPACE_NAME --name NEW_WORKSPACE_NAME``.

**Bug Fixes**

-  Agent: Fixed a bug where in certain cases of the master restarting with active tasks, the agent
   resource manager could prevent other tasks from running.
-  Kubernetes: When a TensorBoard inherits its images from an experiment configuration, it now also
   inherits the ``environment.pod_spec.spec.imagePullSecrets`` value.

Version 0.19.4
==============

**Release Date:** September 22, 2022

**Breaking Changes**

-  ``det deploy aws``: Remove ``--deployment-type=vpc`` option. Please use ``efs`` or ``fsx``
   deployment types instead.

**API Changes**

-  The ``STATE_ACTIVE`` state for experiments and trials is now divided into four sub-states:
   ``STATE_QUEUED``, ``STATE_PULLING``, ``STATE_STARTING``, and ``STATE_RUNNING``. Queries to
   ``GetExperimentsRequest`` that filter by state continue to use ``STATE_ACTIVE``.

-  The possible states of tasks have been adjusted to match those of experiments and trials. The
   previous ``STATE_PENDING`` and ``STATE_ASSIGNED`` are now ``STATE_QUEUED``.

**Bug Fixes**

-  Checkpoints: Fixed a bug where operations that listed checkpoints could sometimes return the same
   checkpoint multiple times.

Version 0.19.3
==============

**Release Date:** September 09, 2022

**Improvements**

-  Slurm: Singularity containers may now use AMD ROCm GPUs.
-  Slurm: Podman V4.0+ is now supported in conjunction with the Slurm job scheduler.
-  Kubernetes: The UID and GID of Fluent Bit logging sidecars may now be configured on a
   cluster-wide basis.

**New Features**

-  Example: Allow training of models that do not fit into GPU memory using DeepSpeed ZeRO Stage 3
   with CPU offloading.
-  Kubernetes: Allow the UID and GID of Fluent Bit logging sidecars to be configured on a
   cluster-wide basis.

Version 0.19.2
==============

**Release Date:** August 26, 2022

**Breaking Changes**

-  API: Response format for metrics has been standardized to return aggregated and per-batch metrics
   in a uniform way. ``GetTrialWorkloads``, ``GetTrials`` API response format has changed.
   ``ReportTrialTrainingMetrics``, ``ReportTrialValidationMetrics`` API request format has changed
   as well.

-  API: ``GetJobs`` request format for pagination object has changed. Instead of being contained in
   a nested ``pagination`` object, these are now top level options, in line with the other
   paginatable API requests.

-  CLI: ``det trial describe --json`` output format has changed. Fixed a bug where ``det trial
   describe --json --metrics`` would fail for trials with a very large number of steps.

-  CLI: ``det job list`` will now return all jobs by default instead of a single API results page.
   Use ``--pages=1`` option for the old behavior.

-  The ``/api/v1/trials/:id`` endpoint no longer returns the ``workloads`` attribute. Workloads
   should instead be retrieved from the paginated ``/api/v1/trials/:id/workloads`` endpoint.

**Bug Fixes**

-  Kubernetes: Fixed an issue where restoring a job in a Kubernetes set up could crash the resource
   manager.

-  CLI: Fixed a bug where ``det e set gc-policy`` would fail when deserializing an api response
   because it wasn't adjusted for the new format.

-  Distributed training: Previously, experiments launched with determined.launch.torch_distributed
   were wrongly skipping torch.distributed.run for single-slot trials and invoking training scripts
   directly. As a result, functions such as torch.distributed.init_process_group() would fail, but
   only inside single-slot trials. Now, determined.launch.torch_distributed will conform to the
   intended behavior as a wrapper around torch.distributed.run and will invoke torch.distributed.run
   on all training scripts.

-  Experiments with a single trial are now considered canceled when their trial is canceled or
   killed.

**Improvements**

-  API: ``GetTrialWorkloads`` can now optionally include per-batch metrics when
   ``includeBatchMetrics`` query parameter is set.

**New Features**

-  Cluster: The enterprise edition of Determined (`HPE Machine Learning Development
   <https://www.hpe.com/us/en/hpe-machine-learning-development-environment.html>`_), can now be
   deployed on a Slurm cluster. When using Slurm, Determined delegates all job scheduling and
   prioritization to the Slurm workload manager. This integration enables existing Slurm workloads
   and Determined workloads to coexist and access all of the advanced capabilities of the Slurm
   workload manager. The Determined Slurm integration can use either Singularity or Podman for the
   container runtime.

Version 0.19.1
==============

**Release Date:** August 11, 2022

**Fixes**

-  Fix the Python SDK with Determined 0.19.0. An important endpoint broke in the 0.19.0 release,
   causing several Python SDK methods to break. Additional tests have been added to prevent similar
   breakages in the future.

**Improvements**

-  API: new ``on_training_workload_end`` and ``on_checkpoint_upload_end`` ``PyTorchCallback``
   methods available for use with ``PyTorchTrial`` and ``DeepSpeedTrial``.
-  API: ``PyTorchTrial`` and ``DeepSpeedTrial`` callback ```on_checkpoint_end`` deprecated in favor
   of ``on_checkpoint_write_end``, re-named for clarity.

**New Features**

-  Web: Add a button to start a hyperparameter search experiment based on an experiment or trial.
   The button brings up a form allowing users to change searcher settings and hyperparameter ranges.

Version 0.19.0
==============

**Release Date:** July 29, 2022

**New Features**

-  Introduce a file system cache for model definition files, configured via ``cache.cache_dir`` in
   the master configuration. The default path is ``/var/cache/determined``. Note that the master
   will crash on startup if the directory does not exist and cannot be created.

**Improvements**

-  Security: Setting ``registry_auth.serveraddress`` will now only send credentials to the server
   configured. Not setting ``registry_auth.serveraddress`` is now deprecated when ``registry_auth``
   is set. In the future, ``serveraddress`` will be required whenever ``registry_auth`` is set.

-  Agent: Users may now run ``docker login`` on agent host machines to authenticate with Docker
   registries. Note that if the agent is running inside a Docker container then
   ``~/.docker/config.json`` will need to be mounted to ``$HOME/.docker/config.json`` (by default
   ``/root/.docker/config.json``) inside the container.

-  CLI: The Determined CLI now supports reading a username and password from the ``DET_USER`` and
   ``DET_PASS`` environment variables to avoid the need to run ``det user login``, allowing for
   easier use of the CLI in scripts. ``det user login`` is still the preferred mechanism for most
   use cases of the CLI.

**Breaking Changes**

-  Experiment: The default value for the ``average_training_metrics`` experiment configuration
   option has been changed to ``true``. This change only affects distributed training. The previous
   default of ``false`` leads to only the chief worker's training metrics being reported. Setting
   this configuration to ``true`` instead reports the true average of all workers' training metrics
   at the cost of increased communication overhead. Users who do not require accurate training
   metrics may explicitly set the value to ``false`` as an optimization.

-  API: The ``/projects/:id/experiments`` endpoint has been removed and replaced with a
   ``project_id`` parameter on the ``/experiments`` endpoint.

-  API: The ``config`` attribute in the response of the ``/experiments/:id`` endpoint has been moved
   into the ``experiment`` object. The ``config`` attribute is now also available for experiments
   returned from the ``/experiments`` endpoint.

**Bug Fixes**

-  When creating a test experiment, the container storage path was not being set correctly.

-  Notebooks: Fix a bug where notebooks would ignore the ``--template`` CLI argument.

-  Notebooks: Fix a bug where running ``det notebook start --preview`` would launch a notebook
   instead of just displaying the configuration.

-  Kubernetes: Fix an issue where zero-slot tasks would use the GPU image instead of the CPU image.

-  Kubernetes: Fix an issue where zero-slot tasks would incorrectly be exposed to all GPUs.

-  Kubernetes: Fix an issue where the Helm option ``defaultPassword`` caused the deployment to hang.

-  Ensure an allocation's recorded end time is always valid, even on restoration failures. Invalid
   end times could cause historical reporting rollups to fail. If there were any failures, they will
   be fixed by database migrations this update.

**Security Fixes**

-  **Breaking Change** PyTorch Lightning is no longer a part of Determined environments. When
   needed, it should be installed as part of startup hooks.

**************
 Version 0.18
**************

Version 0.18.4
==============

**Release Date:** July 14, 2022

**New Features**

-  Configuration: Add support for ``task_container_defaults.environment_variables`` in the master
   config, which allows users to specify a list of environment variables that will be set in the
   default task container environment.

-  Web: Most user settings and preferences, like filters, are now persisted to the database. Users
   will now be able to retain their settings across devices.

**Bug Fixes**

-  Since 0.17.7, ``det experiment download-model-def $ID`` has been saving the downloaded tarballs
   as just ``$ID``. This release corrects that behavior and names them
   ``experiment_$ID_model_def.tgz`` instead.

-  Kubernetes: Fix a bug where following the link to live TensorBoards would redirect to the
   ``Uncategorized`` page.

-  Ensure an allocation's recorded end time is always valid, even on restoration failures. Invalid
   end times could cause historical reporting rollups to fail. Previous failures, if any, will be
   fixed by database migrations this update.

**Improvements**

-  Add the resource pool field when listing experiments or commands in Kubernetes, where it was
   previously left blank.

Version 0.18.3
==============

**Release Date:** July 07, 2022

**Breaking Changes**

-  WebUI: Remove previously unlisted cluster page. This page has been replaced by a new version
   available through the navigation bar.

**New Features**

-  Workspaces & Projects: Teams can now organize related experiments into projects and workspaces.
   See `video <https://www.youtube.com/watch?v=zJP7p0CWubw>`_ for a walkthrough.

-  Logging: Master configuration now supports ``logging.additional_fluent_outputs`` allowing
   advanced users to specify custom integrations for task logs.

-  Kubernetes: Task init containers no longer require root privileges.

-  API: Trial API now uploads profiling data to the checkpoint storage from all workers. Core API
   users can now pass a new optional argument, ``tensorboard_mode``, to ``core.init()``. The default
   value is ``AUTO``. In ``AUTO`` mode, TensorBoard metrics are written on the chief, and metrics as
   well as profiling data are uploaded to checkpoint storage from the chief only. In ``MANUAL``
   mode, the user is responsible for writing TensorBoard metrics and uploading profiling data. In
   order to make that possible, two new methods are introduced on
   :class:`~determined.core.TrainContext`:
   :meth:`~determined.core.TrainContext.get_tensorboard_path()` returns the path to the directory
   where metrics can be written and :meth:`~determined.core.TrainContext.upload_tensorboard_files()`
   uploads metrics and other files, such as profiling data, to checkpoint storage.

-  Add support for recovering live commands, notebooks, TensorBoards, and shells on master restart.
   This is an extension of live trial recovery, available since version 0.18.1.

**Bug Fixes**

-  WebUI: Fix a bug where a previous resource pool selection would not update when a new resource
   pool is selected for viewing associated jobs.
-  API: Fix a bug where ``/api/v1/tasks/{taskId}`` would often return incorrect allocation states.
-  Since 0.17.15, there was a bug where ``task_container_defaults.registry_auth`` was not correctly
   passed to tasks, resulting in tasks being unable to pull images.

**Improvements**

-  CLI: Add new flag ``--agent-config-path`` to ``det deploy local agent-up`` allowing custom agent
   configs to be used.
-  CLI: Add ``det (notebook|shell|tensorboard) list --json`` option, allowing user to get
   JSON-formatted notebook, shell or TensorBoard task list.
-  Configuration: Experiment configuration ``resources.shm_size`` now supports passing in a unit
   like ``4.5 G`` or ``128MiB``.

Version 0.18.2
==============

**Release Date:** June 14, 2022

**Bug Fixes**

-  Web: Update task cards to only truncate task UUIDs and leave experiment IDs alone.
-  CLI: Fix an issue for ``det task logs`` where trial task IDs and checkpoint GC task IDs could not
   be used.
-  Agent: Fix being unable to use control-C to cancel the agent when it is connecting to master.
-  Trial: Fix a bug where the rendezvous timeout warning could be printed erroneously.
-  Commands: Fix an issue for commands where setting an environment variable as ``FOO`` instead of
   ``FOO=bar`` in ``environment.environment_variables`` causes the agent to panic.

**Fixes**

-  Prevent certain hangs when using one of Determined's built-in launchers, which begin in release
   0.18.0. These hangs were caused by wrapper processes seeing SIGTERM but not passing it to their
   child process.

-  Supports running in containers that do not have a /bin/which path, such as python-slim. The error
   was caused by accidentally hardcoding ``/bin/which`` instead of letting the shell find ``which``
   on the path.

-  Automatically add a ``determined_version`` key to the metadata of checkpoints created by any of
   the Trial APIs. This automatic key was accidentally dropped in release 0.18.0. Note that Core API
   checkpoints have full control over their checkpoint metadata and so are unaffected.

**Improvements**

-  Scheduler: Tasks now release resources as they become free instead of holding them until all
   resources are free.
-  CLI: ``det deploy aws up``, ``det deploy aws down``, and ``det deploy gcp down`` now take
   ``--yes`` to skip prompts asking for confirmation. ``--no-prompt`` is still usable.
-  Experiments: When attempting to delete an experiment, if the delete fails it is now retryable.
-  Agents: Improve behavior and observability when agents lose WebSocket messages due to network
   failures.
-  Trials: Trial logs will report some system events such as when a trial gets canceled, paused,
   killed, or preempted.

**New Features**

-  Kubernetes: Specifying ``observability.enable_prometheus`` in Helm will now correctly enable
   Prometheus monitoring routes.

-  Kubernetes: Users may now specify a ``checkpointStorage.prefix`` in the Determined Helm chart if
   using S3 buckets for checkpoint storage. Checkpoints will now be uploaded with the path prefix
   whereas before it was ignored.

-  CLI: Add new command ``det experiment logs <experiment-id>`` to get logs of the first trial of an
   experiment. Flags from ``det trial logs`` are supported.

-  Configuration: Add support for ``checkpointStorage.prefix`` in master and experiment
   configuration for Google Cloud Storage (``gcs``).

**Security Fixes**

-  API: Endpoints under ``/debug/pprof`` now require authentication.

Version 0.18.1
==============

**Release Date:** May 24, 2022

**New Features**

-  Web: Themes have been introduced and styles have been adjusted to support various themes. Theme
   switching is currently limited to dark/light mode and is set first through OS-level preferences,
   then through browser-level preference. In-app controllers will be coming soon.

-  Add experimental support for recovering live trials on master process restart. Users can restart
   the master (with updated configuration options or an upgraded software version), and the current
   running trials will continue running using the original configuration and harness versions. This
   requires the agent to reconnect within a configurable ``agent_reconnect_wait`` period. This is
   only available for the ``agent`` resource manager, and can be enabled for resource pools using
   the ``agent_reattach_enabled`` flag. May only be available for patch-level releases.

-  Web: A trial restart counter has been added to the experiment detail header for single-trial
   experiments. For multi-trial experiments, trial restart counts are shown in a new `Restarts`
   column in the `Trials` table.

   .. image:: https://user-images.githubusercontent.com/220971/169450333-c3dde9f4-abc0-4f8b-9e83-216e13ee2ca0.png
      :alt: Trial restart counter

   .. image:: https://user-images.githubusercontent.com/220971/169450323-d169f4ee-2698-4ae8-9b1a-c04460751310.png
      :alt: Restarts column in the Trials table

**Improvements**

-  Security: Improved security by requiring admin privileges for the following actions.

   -  Reading master config.
   -  Enabling or disabling an agent.
   -  Enabling or disabling a slot.

-  Logging: Ensure logs for very short tasks are not truncated in Kubernetes.

-  Web: Centralize sidebar options ``Cluster``, ``Job Queues``, and ``Cluster Logs`` into
   ``Cluster`` page for a simplified layout.

-  Web: In order to provide a more precise view of resource pools, new fields like ``accelerator``
   and ``warm slots`` have been added.

-  Web: Clicking on resource pool cards will lead to a detail page, which also includes a ``Stats``
   tab showing average queued time by day.

**Breaking Changes**

-  Security: The following routes and CLI commands now need admin privileges.

   -  ``/config``
   -  ``/api/v1/master/config``
   -  ``/api/v1/agents/:agent_id/enable``
   -  ``/api/v1/agents/:agent_id/disable``
   -  ``/agents/:agent_id/slots/:slot_id``
   -  ``/api/v1/agents/:agent_id/slots/:slot_id/enable``
   -  ``/api/v1/agents/:agent_id/slots/:slot_id/disable``
   -  ``det master config``
   -  ``det agent enable``
   -  ``det agent disable``
   -  ``det slot enable``
   -  ``det slot disable``

-  Logging: The default Fluent Bit version in all deployment modes is now 1.9.3, changed from 1.6.

**Bug Fixes**

-  Web: Fix the user filtering for migrating from Determined `0.17.15` to Determined `0.18.0`.
-  API: Fix an issue where the ``POST /users`` endpoint always returned an error instead of the
   user's information, even when the user was created successfully.

Version 0.18.0
==============

**Release Date:** May 09, 2022

**New Features**

-  Add the Core API. The Core API is the first API offered by Determined that allows users to fully
   integrate arbitrary models and training loops into the Determined platform. All of the features
   offered by the higher-level Trial APIs, such as reporting metrics, pausing and reactivating,
   hyperparameter search, and distributed training, are now available to arbitrary models,
   frameworks, and training loops, with only light code changes.

-  **Breaking Change**: Checkpoints: The Python SDK's ``Checkpoint.download()`` method now writes a
   differently formatted ``metadata.json`` file into the checkpoint directory. Previously, the JSON
   content in the file contained many system-defined fields, plus a ``metadata`` field that
   contained the user-defined metadata for the checkpoint, which was also available as a Python
   object as ``Checkpoint.metadata``. Now, ``metadata.json`` contains only the user-defined
   metadata, and those metadata appear as top-level keys. Some of the fields which were previously
   system-defined are now considered user-defined, even though they are uploaded automatically in
   Trial-based training. This decision is in line with the Trial APIs now being optional---that is,
   part of userspace---after the release of the Core API.

-  Job queue: Add support for dynamic job modification on Kubernetes using the job queue. Users can
   now use the WebUI or CLI to change the priority and queue position of jobs in k8s. To update jobs
   through the WebUI, go to the Job Queue section, find the target job, and click on the Manage Job
   option. To update jobs in the CLI, use the ``det job update`` command. Run ``det job update
   --help`` for more information.

**Bug Fixes**

-  CLI: API requests executed through the Python bindings have been erroneously using the SSL
   "noverify" option since version 0.17.6, making them potentially insecure. The option is now
   disabled.

**Deprecated Features**

-  The Determined Data Layer has been deprecated and will be removed in a future version. New code
   should not begin using it, but we will assist existing users to migrate to using `YogaDL
   <https://yogadl.readthedocs.io/en/latest/>`__ directly before removing the feature.

**Removed Features**

-  Python API: The old experimental namespace methods for custom reducers in both PyTorchTrial and
   EstimatorTrial have been removed. The experimental names were deprecated in 0.15.2 (April 2021)
   when custom reducers were promoted to general availability. Any users who have not already
   migrated to the non-experimental namespace for custom reducer methods must do so.

-  Searcher: Remove the PBT searcher, which was deprecated in version 0.17.6 (January 2022).

-  API: Remove the notebook logs endpoint in favor of the new task logs endpoint.

-  Python API: Remove the remaining parts of the Native API, which was deprecated in version 0.13.5
   (September 2020). The only Native API functions that still remained were
   ``det.experimental.create()`` and ``det.experimental.create_trial_instance()``.

-  Python API: Remove the ``det.pytorch.reset_parameters()`` function, which was deprecated in
   0.12.13 (August 2020).

**************
 Version 0.17
**************

Version 0.17.15
===============

**Release Date:** April 22, 2022

**Breaking Changes**

-  API: Endpoints for getting or updating a user now accept a ``userId`` instead of ``username`` as
   the path parameter.

**Bug Fixes**

-  Fix an issue where deleted experiments would get stuck in a ``DELETING`` state indefinitely due
   to their checkpoint GC tasks not completing.

-  API: Fix an issue where a reported job state could be stale due to a faulty caching mechanism.
   This could have resulted in an experiment showing in `queued` or `scheduled` state, either in CLI
   or WebUI, when it was in the other state.

**New Features**

-  Add a translation of DeepSpeed's DCGAN example using the new DeepSpeedTrial API.

Version 0.17.14
===============

**Release Date:** April 13, 2022

**Bug Fixes**

-  Resource Pool: Fix a bug that causes the resource pool and resource manager to crash after
   submitting a command with a non-default priority. We recommend that all users on 0.17.12 and
   0.17.13 update to 0.17.14 or later.

Version 0.17.13
===============

**Release Date:** April 07, 2022

**New Features**

-  Support DeepSpeed with a new DeepSpeedTrial API.

   `DeepSpeed <https://www.deepspeed.ai/>`__ is a powerful library for training large scale models.
   With the new ``DeepSpeedTrial`` you can combine all the benefits of Determined with the features
   available in DeepSpeed like the Zero Redundancy Optimizer and pipeline parallel training. We also
   provide an example based on Eleuther AI's `GPT-NeoX <https://github.com/EleutherAI/gpt-neox/>`__
   repo to help you get started training state-of-the-art language models.

-  CLI: Allow the CLI to accept any unique prefix of a task UUID to refer to the task, rather than
   requiring the entire UUID. In some places, Determined only displays the first few characters of a
   UUID.

**Improvements**

-  Model Hub: add support for panoptic segmentation.

   -  Model Hub mmdetection now supports panoptic segmentation task in addition to object detection.
      Previously, the associated Docker image lacked dependencies for panoptic segmentation. Users
      can now use mmdetection configs under ``panoptic_fpn`` and also the ``coco_panoptic`` dataset
      base config.

-  Collect data for agent/instance start time and end time in order to track unused GPUs. Two new
   ``kinds`` (``agent`` and ``instance``) added to CSV report at Cluster page.

**API Changes**

-  The model registry API now accepts either the ID or model name in ``/api/v1/models/:id`` or
   ``/api/v1/models/:name``. This applies to all API routes for models and model versions.
-  The ID can be used in the API and the WebUI (``/det/models/:id``) as a permanent link to the
   model.

**Breaking Changes**

-  Changed the message body of PatchModelRequest and PatchModelVersionRequest such that the POST-ed
   body is the PatchModel or PatchModelVersion object, instead of being wrapped in ``{ "model":
   PatchModel }``.

-  Updated typing hints on other Model Registry API endpoints to make it clear which fields will be
   returned in API responses.

**Bug Fixes**

-  Fix an issue where the originally requested page to redirect to after a previously successful
   authentication flow was not remembered.
-  Fix an issue where trial logs may display timestamps twice.

Version 0.17.12
===============

**Release Date:** March 28, 2022

**New Features**

-  Job queue: Add support for dynamic job modification using the job queue. Users can use the WebUI
   or CLI to change the priority, weight, resource pool, and queue position of jobs without having
   to cancel and resubmit them. This feature is currently available for the fair share and priority
   schedulers. To update jobs through the WebUI, go to the **Job Queue** section and find the
   **Manage Job** option for a job. To update jobs using the CLI, use the ``det job update``
   command. Run ``det job update --help`` for more information.

**Breaking Changes**

-  API: Remove these legacy endpoints:

   -  ``/:experiment_id``
   -  ``/:experiment_id/checkpoints``
   -  ``/:experiment_id/config``
   -  ``/:experiment_id/summary``
   -  ``/:experiment_id/metrics/summary``
   -  ``/:trial_id/details``
   -  ``/:trial_id/metrics``

   The data from those endpoints are still available through the new REST API endpoints under the
   ``/api/v1/experiments/:experiment_id`` and ``/api/v1/trials/:trialᵢd`` prefixes.

**Improvements**

-  Images: Update default environment images to PyTorch 1.10.2, TensorFlow 2.8, and Horovod 0.24.2.

**Bug Fixes**

-  Database migrations: Ensure that migrations run in transactions. The lack of transactional
   migrations surfaced as a bug where, if the master was restarted during a migration, it would
   attempt to rerun the migration when it was already partially or wholly applied (but not marked as
   complete), resulting in various SQL errors on non-idempotent DDL statements.

-  Distributed training: Allow multiple ranks within a distributed training job to report invalid
   hyperparameter exits. Previously, if more than one report was received, the experiment would
   fail.

Version 0.17.11
===============

**Release Date:** March 14, 2022

**New Features**

-  Add ``on_trial_startup()`` and ``on_trial_shutdown()`` methods to
   :class:`~determined.pytorch.PyTorchCallback`. Whenever ``on_trial_startup()`` is called,
   ``on_trial_shutdown()`` is always called before the trial container shuts down. These callbacks
   make it possible to do reliable resource management in a training container, such as if you wish
   to start a background thread or process for data loading and shut it down before the process
   exits.

Version 0.17.10
===============

**Release Date:** March 03, 2022

**Breaking Change**

-  API: PyTorch Lightning has been updated from 1.3.5 to 1.5.9 to address a security vulnerability.
   Experiments using PyTorch Lightning Adapter with v1.3.5 are no longer supported.

**New Features**

-  Added PyTorch example using Bootstrap Your Own Latent (BYOL) to do self-supervised, no labels,
   image classification.

-  PyTorchTrial and TFKerasTrial now automatically log the number of batches and number of records
   in every training and validation workload, as well as the duration of the workload and the
   calculated batches per second and records per second to make tracking progress easier.

-  All (non-experiment) task logs are now persisted. Task logs can be retrieved through the new
   ``det task logs`` CLI command, or the WebUI or REST API. Task logs are now accessible even after
   a master restart, or 72 hours post completion.

-  Support specifying root certificates for the DB via the Determined Helm chart. This allows
   Determined to use SSL to connect to the DB without having to replace the master config manually.
   To use this feature, save the certificate in a configmap or secret and set the following values:
   ``sslMode``, ``sslRootCert``, ``resourceType``, and ``certResourceName``. Additional details can
   be found in the default values.yaml file.

Version 0.17.9
==============

**Release Date:** February 11, 2022

**New Features**

-  Python API: Add new framework-specific methods for loading checkpoints:

   -  :meth:`determined.pytorch.load_trial_from_checkpoint_path`
   -  :meth:`determined.keras.load_model_from_checkpoint_path`
   -  :meth:`determined.estimator.load_estimator_from_checkpoint_path`

   These new methods are part of a larger effort to support more frameworks.

-  Python API: Add :meth:`~determined.pytorch.PyTorchCallback.on_training_epoch_end` method to
   :class:`~determined.pytorch.PyTorchCallback`. Add ``epoch_idx`` argument to
   :meth:`~determined.pytorch.PyTorchCallback.on_training_epoch_start`. Overriding
   ``on_training_epoch_start`` without the ``epoch_idx`` argument is still supported for backward
   compatibility, but doing so is discouraged.

-  Web: Add a column picker to the experiment list page to allow users to choose which table columns
   to display.

   .. image:: https://user-images.githubusercontent.com/15078396/152874244-51e0d84a-3678-4427-b082-ccc0c865200f.png
      :alt: Customize columns picker

   .. image:: https://user-images.githubusercontent.com/15078396/152874240-6365b276-3f3e-4fb6-aa2b-0cedc7451b12.png
      :alt: Customize columns picker displaying columns matching search criteria

-  Notebooks: Add a config field ``notebook_idle_type`` that changes how the idleness of a notebook
   is determined for the idle timeout feature. If the value is different from the default, users do
   not need to manually shut down kernels to allow the idle timeout to take effect.

-  Web: Use the `Page Visiblity API
   <https://developer.mozilla.org/en-US/docs/Web/API/Page_Visibility_API>`__ to detect changes in
   page visibility and avoid unnecessary polling, which can be expensive. While the user is not
   actively focused on the page, all polling is stopped; if the page becomes visible again, any
   previously active polling is restarted.

**Improvements**

-  **Breaking Change:** CLI: The ``det master config`` command now takes the ``--json`` and
   ``--yaml`` options to configure its output format, rather than ``-o <output>``.

-  **Breaking Change**: API: The ``/api/v1/preview-hp-search`` endpoint no longer includes units
   (epochs/records/batches) in its response.

-  API: The ``PATCH /api/v1/experiments/:id`` route no longer uses a field mask. When you include a
   field in the body (e.g., notes or labels) that field will be updated, if it is excluded then it
   will remain unchanged.

-  API: When an experiment successfully completes, its progress value will be set to 100% instead of
   0% or null; when an experiment fails, its progress value will stay the same instead of being
   reset to 0% or null.

-  API: Calls to ``/api/v1/experiments`` and ``/api/v1/experiments/:id`` will return a progress
   value of null instead of 0 in cases where the progress has not been recorded or was reset to
   null.

**Deprecations**

-  Python API: ``Checkpoint.load`` is deprecated. It should be replaced by
   :meth:`determined.experimental.client.Checkpoint.download` along with the appropriate one of the
   new framework-specific functions for loading checkpoints.

-  Python API: The following methods on objects in :mod:`determined.experimental.client` are
   formally deprecated (even though they were not technically public methods previously):

   -  ``Model.from_json``
   -  ``Checkpoint.from_json``
   -  ``Checkpoint.parse_metadata``
   -  ``Checkpoint.get_type``

   These methods will be removed in a future version.

**Removed Features**

-  API: Remove ``/searcher/preview``, ``/checkpoints``, and ``/checkpoints/:checkpoint_id/*``
   endpoints from the legacy API. These functions were already replaced by the gRPC API
   (``/api/v1/preview-hp-search`` and ``/api/v1/checkpoints``) in the web UI, CLI, and tests.

Version 0.17.8
==============

**Release Date:** February 3, 2022

**Bug Fixes**

-  Distributed Training: Fix a bug that shows experiments in a COMPLETED state even if they errored
   out. We recommend that users of distributed training update to 0.17.8 or later.

Version 0.17.7
==============

**Release Date:** January 26, 2022

**Breaking Changes**

-  API: Routes with ``/api/v1/models/:id/*`` are replaced by ``/api/v1/models/:name/*``. Spaces and
   special characters in a name must be URI-encoded. You can get a model by ID with
   ``/api/v1/models?id=<id>``.

-  API: On the list of models (``/api/v1/models``) the optional name parameter is now a
   case-sensitive match, unless you add the parameter ``name_case_insensitive=true``.

-  Python API: :meth:`determined.experimental.client.Determined.get_model` now takes a name rather
   than an ID. Use :meth:`determined.experimental.client.Determined.get_model_by_id` to get a model
   from its ID.

-  Model Registry: New model names must not be blank, have a slash, have multiple spaces, only
   numbers, or be case-insensitive matches to an existing model name.

-  Model Registry: Model names with a forward slash will replace the slash in the name with '--'.

**Bug Fixes**

-  Master: Fix a bug in the priority scheduler where jobs with equal priority would be scheduled or
   preempted in an order not correctly respecting job submission time.

**Removed Features**

-  API: remove ``/experiment-list``, ``/experiment-summaries``, and ``/:experiment_id/kill``
   endpoints from the legacy API. These functions are now replaced by the gRPC API
   (``/api/v1/experiments``) in the web UI, CLI, and tests.

Version 0.17.6
==============

**Release Date:** January 20, 2022

**New Features**

-  Master: Add support for `systemd socket activation
   <https://0pointer.de/blog/projects/socket-activation.html>`__ to the master.

-  Scheduling/CLI: Add support for adjusting job priority and weight through the WebUI and CLI.

-  Add experimental ROCm support. In the environment config for images and environment variables,
   the ``rocm`` key configures ROCm support. The ``gpu`` key has been renamed to ``cuda``; ``gpu``
   is still supported for backward compatibility, but its use is discouraged.

**Improvement**

-  Docs: Improve many pages to address onboarding gaps.

**Bug Fixes**

-  Master: Fix an issue where an update to an experiment's name wouldn't be reflected in its job
   representation until a master restart.
-  Agent: Fix displayed CPU core count for CPU slots.
-  WebUI: Fix an issue where the JupyterLab modal didn't pass the full config.
-  WebUI: Fix the issue of the profiler filter UI not triggering updates.

**Improvements**

-  Logging: Decrease the volume of Docker image pull logs that are rendered into trial logs, and
   make the overall image pull progress more understandable by combining all layers' progress into a
   single progress bar.

**Deprecated Features**

-  Searcher: The Population Based Training searcher (``pbt`` in the searcher config) will be removed
   in the next release.
-  Model Registry: The API and Python interface will be returning to primarily identifying models
   based on their names, rather than their numeric IDs, in the next release.

**Removed Features**

-  Remove support for Python 3.6, which has reached end-of-life.

Version 0.17.5
==============

**Release Date:** December 10, 2021

**New Features**

-  Add reporting of job queue state. The ordering of jobs in the queue and their status can be
   viewed through Determined WebUI, and CLI.

-  WebUI: Add buttons to the WebUI to create new models in the Model Registry, as well as add
   checkpoints as versions to existing models. The Register Checkpoint modal can be accessed through
   the Checkpoint modal. The New Model modal can be accessed through the Register Checkpoint modal
   or on the Model Registry page.

   .. image:: https://user-images.githubusercontent.com/15078396/144926870-bb93d587-f7ad-4052-a338-6fc000bd2ed9.png
      :alt: Model Registry page

   .. image:: https://user-images.githubusercontent.com/15078396/144926881-98aeb187-aa3f-4e40-b502-d7af624573db.png
      :alt: Register Checkpoint page

   .. image:: https://user-images.githubusercontent.com/15078396/144926889-eec0216a-dacc-4fe5-ac28-858ea6587d04.png
      :alt: Create Model page

-  API: Add a method for listing trials within an experiment.

**Improvements**

-  Agent: Improve handling of master connection failures.

**Bug Fixes**

-  Deploy: Fix a bug where GCP clusters created with ``--no-filestore`` still had unused filestores
   created.

Version 0.17.4
==============

**Release Date:** November 30, 2021

**New Features**

-  WebUI: Add the :ref:`model registry <organizing-models>` as a new top-level navigation option,
   allowing for viewing, editing, and deleting existing models created using the CLI.

-  Add experimental support for `Multi-Instance GPUs
   <https://www.nvidia.com/en-us/technologies/multi-instance-gpu/>`__ (MIGs) to agent-based setups,
   in parity with the experimental support for MIGs in Kubernetes-based setups. Static agents and
   Kubernetes clusters may be able to use MIG instances for some workloads. Distributed training is
   not supported, and all MIG instances and nodes within a resource pool must still be homogeneous.

**Improvements**

-  **Breaking Change**: Model Registry: The names of models in the model registry must now be
   unique. If multiple models were previously created with the same name in the registry, the names
   will change.

-  Model Registry CLI: Allow models to be referred to by their now-unique names, not only by ID.

-  Tasks: Historical usage over users now properly accounts for all task types (commands, notebooks,
   etc.), not just trials.

-  Images: Add environment images for TF 2.7.

-  Agent: The ``environment.force_pull_image: true`` option no longer deletes the environment image
   before re-pulling it. Now, it will only fetch updated layers, which is much less wasteful of
   network resources and execution time.

**Bug Fixes**

-  Master: Fix a bug where deleting experiments with trial restarts always failed, and then failed
   to be marked as failed.

Version 0.17.3
==============

**Release Date:** November 12, 2021

**Improvements**

-  Model Registry APIs: Add PATCH and DELETE endpoints to update the attributes of models and model
   versions.
-  Model Registry: Allow models to be deleted only by the user who created them.
-  Security and Logging: When a job is run on Kubernetes as a non-root user, the corresponding
   Fluent Bit sidecar will also run as a non-root user.
-  Deploy: ``det deploy`` will now confirm potentially destructive updates on AWS unless
   ``--no-prompt`` is specified.

**Bug Fixes**

-  Model Registry APIs: Change the ``/models/{}/versions/{}`` to accept model ID as an int.

Version 0.17.2
==============

**Release Date:** October 29, 2021

**New Features**

-  Model Registry APIs: Add new APIs to create a model with labels and to update the labels of an
   existing model.

**Improvements**

-  **Breaking Change:** Deploy: ``det deploy`` now uses cloud images that use the NVIDIA Container
   Toolkit on agent hosts instead of relying on an older NVIDIA runtime, and custom images should be
   updated to do the same. Determined will no longer override the default container runtime
   according to the workload.

-  **Breaking Change:** Model Registry APIs: Require name in the body rather than the URL for the
   ``post_model`` endpoint.

-  **Breaking Change:** Model Registry APIs: Use model ID (integer) instead of name (string) as the
   lookup parameter for the ``get_model`` and ``get_model_versions`` endpoints.

-  Docs: Switch to the `Furo <https://pradyunsg.me/furo/>`__ Sphinx theme, which fixes searching in
   the docs.

**Bug Fixes**

-  Model Registry APIs: Sort models by name, description, and other attributes.
-  Harness: Represent infinite and NaN metric values as strings in JSON.
-  WebUI: Convert infinite and NaN value strings to numeric metrics.
-  WebUI: Report login failures caused by the cluster being unreachable.

Version 0.17.1
==============

**Release Date:** October 18, 2021

**New Features**

-  WebUI: Add a "Notes" tab allowing for the input and viewing of free-form Markdown text on
   experiment pages. This works for both single-trial experiments and trials within a multi-trial
   experiment.

   .. image:: https://user-images.githubusercontent.com/15078396/136809928-11c815cc-3751-4908-8c6e-34fef3b9858d.png
      :alt: Notes tab in the WebUI

**Improvements**

-  Docs: reorganize documents to be more user-friendly.

   -  Merge some how-to guides, topic guides, and reference guides. Users should now need to read
      very few documents to understand what they need to do in Determined rather than having to jump
      around between documents.

   -  Merge most information on best practices into how-to guides so that users find out about best
      practices as soon as they learn how to use something.

   -  Decompose the top-level FAQ document and move different parts of it to relevant pages so that
      users can develop a better expectation of what common issues they might hit.

-  Profiler: ``samples_per_second`` in PyTorch now reflects samples across all workers.

-  Database migrations: Run upgrades in transactions to improve stability.

**Bug Fixes**

-  Deploy: Fix an issue where the default checkpoint storage directory was not created for some
   users.

Version 0.17.0
==============

**Release Date:** September 28, 2021

**Breaking Changes**

-  Deploy: Remove ``--auto-bind-mount`` support from ``det deploy local``. The new
   ``--auto-work-dir`` feature should be a strictly better experience. Users who depended on the
   ``shared_fs`` directory created by ``--auto-bind-mount`` can implement the same behavior by
   calling ``det deploy local cluster_up`` with a ``--master-config-path`` pointing to a
   ``master.yaml`` file containing the following text:

   .. code:: yaml

      task_container_defaults:
        bind_mounts:
          container_path: ./shared_fs
          host_path: /path/to/your/HOME/dir

-  Deploy: This version of ``det deploy`` will not be able to deploy previous versions of
   Determined. If you need to deploy an older version, please use a matching version of the
   ``determined`` package.

-  Experiment: Include ``maxval`` in ``int``-type hyperparameter ranges. Previously, the docs said
   that the endpoints of the hyperparameter were both inclusive, but in reality the upper limit
   ``maxval`` was never actually selected.

   The reproducibility of hyperparameter selection may differ between Determined v0.16.5 and v0.17.0
   for hyperparameter searches containing ``int``-type hyperparameters as a result of this fix.
   However, the reproducibility of model training for any given set of hyperparameters should be
   unaffected.

-  API: Endpoints no longer return the start times of workloads (training, validation, and
   checkpoints). This is part of a longer move to model metrics and workloads separately as part of
   the upcoming generic API.

-  CLI: ``det master config`` now outputs YAML instead of JSON by default. To obtain the old
   behavior, run ``det master config -o json``.

**New Features**

-  Notebooks/TensorBoards: Support a configurable timeout field ``idle_timeout`` that will cause
   notebook and TensorBoard instances to automatically shut down after a period of idleness. A
   notebook is considered to be idle if no kernels or terminals are running and there is no network
   traffic going to the server. A TensorBoard is considered to be idle if there is no network
   traffic going to the server. Note that if you open a notebook file it might open a kernel for
   you, and the kernels and the terminals will not be shut down automatically. You need to manually
   shut down the kernels to make the idle timeout effective.

-  Deploy: Add a new ``--auto-work-dir`` feature to ``det deploy local``. Setting ``--auto-work-dir
   /some/path`` will have two effects: first, ``/some/path`` will be bind-mounted into the container
   (still as ``/some/path``); second, interactive jobs (notebooks, shells, and commands) will run in
   the provided working directory by default. Note that containers run as the root user by default,
   so you may want to :ref:`configure your user <run-as-user>` with ``det user`` such that
   interactive jobs run as your regular user.

-  Commands/shells/notebooks: Support configuring the working directory using the ``work_dir``
   configuration field for commands, shells, and notebooks. You can also optionally set it in the
   ``task_container_defaults.work_dir`` field of the master configuration. The value set in the
   master configuration will be ignored when a context directory is submitted.

-  WebUI: Allow experiment owners to delete their own experiments, singly or in batches.

   .. image:: https://user-images.githubusercontent.com/220971/134048799-cd663a75-cb24-4f44-9a8a-c2ff23222cef.png
      :alt: WebUI showing Delete action

   .. image:: https://user-images.githubusercontent.com/220971/133659677-aea0d1bc-95ce-4652-8218-92b97d114358.png
      :alt: WebUI showing action dropdown selector

-  WebUI: Display the latest log entry available for a trial at the bottom of the trial's page. This
   works for both single-trial experiments and trials within a multi-trial experiment.

   .. image:: https://user-images.githubusercontent.com/220971/131391658-4be1a1f4-1d46-4766-a737-7eb8efcb65b4.png
      :alt: WebUI displaying the latest log entry

-  WebUI: Add support for displaying NaN and Infinity metric values.

-  Model Hub: Support the `MMDetection <https://github.com/open-mmlab/mmdetection>`__ library to
   easily train object detection models. MMDetection provides efficient implementations of popular
   object detection methods like Mask R-CNN, Faster R-CNN, and DETR on par with Detectron2. In
   addition, cutting-edge approaches from academia are regularly added to the library.

-  Deploy: Add the ability to use customizable master configuration templates in ``det deploy
   aws|gcp``.

-  Images: Add an environment image for CPU-only TensorFlow 2.5 and 2.6.

**Improvements**

-  API: The aggregated historical resource allocation APIs
   ``/api/v1/resources/allocation/aggregated`` and ``/allocation/aggregated`` now account for all
   resources, not just those allocated to experiments.

-  Images: Add CPU-only images for TF 2.5 and 2.6.

-  Images: Upgrade JupyterLab to version 3.1.

-  Images: TF 2.5 and 2.6 images will no longer include PyTorch builds. For PyTorch 1.9, please use
   the combined TF 2.4/PyTorch 1.9 image.

-  Images: TF 2.4, 2.5, 2.6, and PyTorch 1.9 images will now use Python 3.8. The legacy TF
   1.15/PyTorch 1.7 image will continue to use Python 3.7.

**Changes**

-  WebUI: Change the task list page to open new tabs when user clicks on task links.
-  WebUI: The trial detail page will no longer show workload-based start time information, including
   training time, validation time, and checkpoint time.

**Bug Fixes**

-  WebUI: Fix continuing trials with nested hyperparameters.

**************
 Version 0.16
**************

Version 0.16.5
==============

**Release Date:** September 3, 2021

**New Features**

-  Support custom PyTorch data loaders with ``PyTorchTrial``. You may now call
   :meth:`context.experimental.disable_dataset_reproducibility_checks()
   <determined.pytorch.PyTorchExperimentalContext.disable_dataset_reproducibility_checks>` in your
   trial's ``__init__()`` method, which will allow you to return arbitrary ``DataLoader`` objects
   from :meth:`~determined.pytorch.PyTorchTrial.build_training_data_loader` and
   :meth:`~determined.pytorch.PyTorchTrial.build_validation_data_loader`. This is desirable when
   your data loader is not compatible with Determined's ``det.pytorch.DataLoader``. The usual
   dataset reproducibility that ``det.pytorch.DataLoader`` provides is still possible to achieve,
   but it is your responsibility. You may find the ``Sampler`` classes in
   :mod:`determined.pytorch.samplers` to be helpful.

**Improvements**

-  Add the ability to disable agents while allowing currently running tasks to finish using ``det
   agent disable --drain AGENT_ID``.

**Bug Fixes**

-  WebUI: Show metrics with a value of 0 in graphs.
-  Properly load very old (pre-0.13.8) checkpoints with ``TFKerasTrial``.

Version 0.16.4
==============

**Release Date:** August 23, 2021

**New Features**

-  WebUI: Add a trial comparison modal, allowing comparison of information, metrics, and
   hyperparameters between specific trials within an experiment. This is available from the
   experiment trials and experiment visualization pages.

-  Scheduling/CLI: Support changing task priorities using the ``det
   experiment/command/notebook/shell/tensorboard set priority`` commands.

-  CLI: Allow command-line config overrides in experiment creation, e.g., ``det e create const.yaml
   . --config key=value``.

-  WebUI: Allow cluster admins to delete individual experiments.

**Bug Fixes**

-  Cluster: Fix breakage in trial fault tolerance caused by not sending enough state snapshots.
-  WebUI: Prevent logs from potentially introducing harmful HTML/JS injections via Unicode.
-  WebUI: Change y-axis of the profiler timing metrics chart from milliseconds to seconds.
-  WebUI: Prevent the zoom from resetting when chart data series are added.
-  WebUI: Fix the issue of learning curves not resizing properly.

Version 0.16.3
==============

**Release Date:** July 22, 2021

**New Features**

-  Add the ability to use Azure Blob Storage for checkpoint storage.
-  Add support for Azure Kubernetes Service, including updating Helm to support Azure Blob Storage
   and adding additional docs for AKS.
-  WebUI: Add support for nested hyperparameters in experiment config, trial hyperparameters, and
   hyperparameter visualization.
-  WebUI: Add the ability to view trial logs and open TensorBoards directly from the trial list
   view.
-  WebUI: Enable sorting and filtering trials by state on an experiment's trials page.
-  WebUI: Add a server availability check on load.

**Bug Fixes**

-  Fix a bug where experiments with model definitions exceeding 50% of the maximum allowable size
   would cause trials to never start.
-  WebUI: Prevent hyperparameter visualization from getting stuck showing a spinner after clicking
   through all the different tabs.
-  WebUI: Fix the issue of experiments showing incorrect data if they were forked from another
   experiment or continued from a trial.

Version 0.16.2
==============

**Release Date:** July 9, 2021

**New Features**

-  Make ``det deploy aws up`` automatically bind-mount FSx and EFS directories into task containers
   when available.

-  Make ``det deploy local`` bind-mount the user's home directory into task containers. The mounted
   directory can be changed with the ``--auto-bind-mount=<path>`` option and mounting can be
   disabled entirely with ``--no-auto-bind-mount``.

**Improvements**

-  PyTorchTrial: Improve support for custom batches in PyTorch, e.g., as used in
   ``pytorch_geometric``. See :meth:`~determined.pytorch.PyTorchTrial.get_batch_length` or
   ``examples/graphs/proteins_pytorch_geometric`` for further details.

**Bug Fixes**

-  WebUI: Avoid waiting for an extra polling cycle to load trial data when loading single-trial
   experiments.
-  WebUI: Fix an issue with boolean hyperparameter values not being rendered in learning curve
   tables.

Version 0.16.1
==============

**Release Date:** June 28, 2021

**New Features**

-  Add support for CPU-based training. This makes it possible to run Determined on clusters without
   GPUs, including on-prem, AWS, GCP, and Kubernetes-based (default scheduler only) configurations.

-  Support spinning up and down a Filestore instance when running ``det deploy gcp up/down``. The
   Filestore instance will automatically be mounted to agents and bind-mounted into task containers.
   You can also use a pre-existing Filestore instance.

**Improvements**

-  **Breaking Change:** REST API: Rename ``gpu`` and ``cpu`` fields in ``ResourcePool`` object to
   ``compute`` and ``aux``.

-  **Breaking Change:** Deploy: In ``det deploy gcp`` and ``det deploy aws``, rename the default
   compute pool from ``gpu-pool`` to ``compute-pool``. When upgrading a cluster from a previous
   version, existing pending experiments may error out and need to be resubmitted.

**Bug Fixes**

-  Support using Docker images with ``EXPOSE`` commands as images for notebooks/shells/TensorBoards.
   Previously, the ``EXPOSE`` command could break proxying through the Determined master.

Version 0.16.0
==============

**Release Date:** June 14, 2021

**New Features**

-  Python SDK: Extend the Checkpoint Export API into a Python SDK capable of launching and
   controlling experiments on the cluster directly from Python. See the documentation and examples
   in the :mod:`~determined.experimental.client` module.

-  Trials: Add new support for profiling model code. For all frameworks, collecting system metrics,
   such as GPU utilization and memory, is supported. For PyTorch, additional profiling for timing is
   available. To quickly try out profiling, set ``profiling.enabled = true`` in the experiment
   configuration.

-  Experiments: Add new ``notes`` and ``name`` fields to experiments.

-  REST API: Add new parameters to ``/api/v1/experiments`` to filter and sort experiments by name.

-  Master configuration: Support ``bind_mounts`` in ``task_container_defaults`` in the master
   configuration. The configured directories will be mounted for experiments, notebooks, commands,
   shells, and TensorBoards.

-  Images: Add an environment image containing TensorFlow 2.5 and CUDA 11.2.

**Improvements**

-  **Breaking change**: JupyterLab: Upgrade the JupyterLab version to 3.0.16. JupyterLab will no
   longer work with previously released images. Custom image users should upgrade to JupyterLab 3.0
   or higher.

-  Scheduling: Support backfilling in the priority scheduler. If there are slots that cannot be
   filled with high-priority tasks, low-priority tasks will be scheduled onto them. This requires
   preemption to be enabled in the :ref:`master configuration <master-config-reference>`.

-  WebUI: Improve task list filtering by moving column filters to the table header.

-  REST API: Change filtering experiments by description to be case-insensitive when using the
   ``/api/v1/experiments`` endpoint.

**Bug Fixes**

-  Fix a bug where ``InvalidHP`` exceptions raised in the trial ``__init__()`` caused the trial to
   restart.

-  WebUI: Fix an issue with representing some hyperparameter values as text.

-  Kubernetes: Prevent Determined from sometimes crashing when handling concurrent job submissions.

-  Master configuration: Fix a bug that was triggered when the master configuration had S3 secrets
   explicitly configured in ``checkpoint_storage``. Experiments that did not override the
   master-provided checkpoint storage would fail.

**Deprecated Features**

-  The method :meth:`~determined.experimental.create_trial_instance` is now deprecated. Users should
   instead use the more flexible ``TrialContext.from_config()``, which is described in
   :ref:`model-debug`.

**Removed Features**

-  The methods ``det.experimental.keras.init()`` and ``det.experimental.estimator.init()`` have been
   removed. They were deprecated in 0.13.5.

**************
 Version 0.15
**************

Version 0.15.6
==============

**Release Date:** June 2, 2021

**New Features**

-  Add PyTorch's `word-level language modeling RNN example
   <https://github.com/pytorch/examples/tree/main/word_language_model>`__ as a `Determined example
   <https://github.com/RehanSD/determined/blob/master/examples/nlp/word_language_model/README.md>`__.

-  Support using the Determined shell as a remote host inside Visual Studio Code and PyCharm IDEs.

**Improvements**

-  Deploy: Add support for ``terraform`` 0.15 when using ``det deploy gcp``.

-  REST API: Add a ``preview`` parameter to the Notebook launch API (``POST /api/v1/notebooks``). If
   set, this API will return a full configuration that is populated with the template and Notebook
   configuration.

-  WebUI: Improve Experiment list filtering by moving column filters to the table header.

-  WebUI: Improve the Trial details page by moving hyperparameters, workloads, and logs into
   separate "tabs" on the Trial detail page.

**Bug Fixes**

-  PyTorchTrial: Fix an issue where a DataLoader iterator that uses multiprocessing could cause a
   hang when exiting.
-  WebUI: Prevent TQDM log lines from generating large quantities of whitespace when rendering logs.

Version 0.15.5
==============

**Release Date:** May 18, 2021

**Bug Fixes**

-  Fix an issue where the master would attempt to schedule onto agents that had previously
   disconnected.

**Deprecated Features**

-  Deprecate the ``scheduler`` and ``provisioner`` fields in the master configuration in favor of
   ``resource_manager`` and ``resource_pools``. They will be removed in the next minor release,
   Determined 0.16.0.

Version 0.15.4
==============

**Release Date:** May 12, 2021

**New Features**

-  Model Hub: Publish Determined's Model Hub library to make it easy to train models from supported
   third-party libraries with a Determined cluster. The first library supported in Model Hub is the
   `Hugging Face Transformers Library for NLP <https://huggingface.co/docs/transformers/index>`__.

**Minor Changes**

-  API: Remove redundant APIs for commands, shells, TensorBoards, and notebooks. The CLI now uses
   updated versions of these endpoints; related CLI commands on versions 0.15.4 and beyond are not
   backward-compatible with previous versions of Determined clusters.

-  API: Update the trial detail endpoint (``GET /api/v1/trials/:id``), dropping
   ``prior_batches_processed`` and ``num_inputs`` in favor of ``total_batches``.

Version 0.15.3
==============

**Release Date:** May 5, 2021

**Bug Fixes**

-  Images: Fix GPU support in CUDA 10.2 + TensorFlow 1.15 images.
-  Trials: Update to match ``websockets>= 9.0`` library API change.
-  Trials: Fix a bug that caused trials to panic upon receiving too many rendezvous addresses.

Version 0.15.2
==============

**Release Date:** April 29, 2021

**New Features**

-  Kubernetes: Support priority scheduling with preemption. The preemption scheduler is able to
   preempt experiments when higher priority ones are submitted.

-  APIs: Promote the custom metric reducer APIs for both :ref:`pytorch <pytorch-custom-reducers>`
   and estimators from experimental status to general availability.

-  Resource pools: Support =configuring distinct ``task_container_defaults`` for each resource pool
   configured on the cluster. This can allow different resource pools which may have very different
   hardware to configure tasks in each pool with the correct settings.

**Improvements**

-  Agent: Support configuring the name of the Fluent Bit logging container via the
   ``--fluent-container-name`` option.

-  Docker: Support specifying the ``--devices``, ``--cap-add``, and ``--cap-drop`` arguments to the
   ``docker run`` command. These are configured in an experiment or command/notebook config via
   ``resources.devices``, ``environment.add_capabilities``, and ``environment.drop_capabilities``.
   These settings can combine to allow an experiment to take advantage of cluster hardware not
   previously available to training or notebook task. These configurations are only honored by
   resource managers of type ``agent``, and are ignored by resource managers of type ``kubernetes``.

**Bug Fixes**

-  Agent: Support the ``--fluent-port`` option.

-  PyTorchTrial: Fix learning rate scheduler behavior when used with gradient aggregation.

-  ``PyTorchTrial``'s :meth:`~determined.pytorch.PyTorchTrialContext.to_device` no longer throws
   errors on non-numeric NumPy-like data. As PyTorch is still unable to move such data to the GPU,
   non-numeric arrays will simply remain on the CPU. This is especially useful to NLP practitioners
   who wish to make use of NumPy's string manipulations anywhere in their data pipelines.

-  TFKerasTrial: Fix support for TensorFlow v2.2.x.

-  WebUI: Fix the issue of the WebUI crashing when user selects a row in experiment list page then
   changes the user filter.

-  WebUI: Fix the issue of agents overview on the Cluster page not updating properly when agents
   shutdown.

-  WebUI: Fix the issue of trial logs not rendering properly on Safari 14.

Version 0.15.1
==============

**Release Date:** April 16, 2021

**Bug Fixes**

-  Trials: Fix ``TFKerasTrial`` on TensorFlow 2 with disabled v2 behavior and/or disabled eager
   execution.
-  Master: Fix two issues that caused experiments to not recover successfully on master crashes
   (after upgrading to version 0.15.0).

Version 0.15.0
==============

**Release Date:** April 14, 2021

**New Features**

-  WebUI: Provide historical allocation data on the Cluster page. This page breaks down GPU hours by
   user, label, and resource pool.
-  WebUI: Add a gallery mode to the hyperparameter scatter plot and heatmap visualizaztions to allow
   users to inspect each scatter plot in full detail.

**Improvements**

-  **Breaking Change** CLI: Consolidate ``det`` and ``det-deploy`` executables into the
   ``determined`` package, which now includes all Determined libraries and tools. The
   ``determined-cli``, ``determined-deploy``, and ``determined-common`` packages are now deprecated.

   -  When upgrading from older versions, ``det`` command may break for some users because of
      ``pip`` limitations. Please uninstall outdated packages, and then reinstall Determined.

-  PyTorch: Remove ``cloudpickle`` as a dependency for PyTorch checkpoints. This does not affect
   compatibility of existing checkpoints. This change will improve portability across Python
   versions.

-  Deploy: Move default storage location for checkpoint data in local clusters deployed via ``det
   deploy local`` to an OS-specific user data directory (e.g. ``$XDG_DATA_HOME/determined`` or
   ``~/.local/share/determined`` on Linux, and ``~/Library/Application Support/determined`` on
   macOS). Previously, ``/tmp`` was used. This location can be changed using the
   ``--storage-host-path`` command line flag of ``det deploy local``. If users provide their own
   custom ``master.yaml`` via ``--master-config-path``, the configured ``checkpoint_storage`` in
   ``master.yaml`` will take precedence.

-  Searcher: Remove support for ``adaptive`` and ``adaptive_simple`` searchers which were deprecated
   in Determined 0.13.7.

**Bug Fixes**

-  WebUI: Fix an issue where the metric value occasionally had the word "undefined" prepended.

**************
 Version 0.14
**************

Version 0.14.6
==============

**Release Date:** April 1, 2021

**New Features**

-  REST API: Add a new endpoint to delete experiments. This endpoint is only enabled for admin users
   and deletes all resources associated with an experiment. This includes checkpoint storage,
   TensorBoards, trial logs from all backends and metadata such as history and metrics, stored in
   PostgreSQL.

-  REST API: Add a new endpoint to fetch aggregated historical resource allocation information.

-  CLI: Add new commands ``det resources raw`` and ``det resources aggregated`` to access resource
   allocation information.

-  PyTorch Lightning: Add an adapter to support ``LightningModule`` from PyTorch Lightning in the
   PyTorchTrial API.

**Improvements**

-  Images: The default environment images have been updated to CUDA 10.2, PyTorch 1.8, and
   TensorFlow 1.15.5 with Python 3.7. Previous images are still available but must be specified in
   the experiment or command configuration. It is recommended to validate the performance of models
   when changing CUDA versions as some models can experience significant changes in training time,
   etc.

-  WebUI: Improve the hyperparameter scatter plot and heat map visualizations by adding support for
   showing categorical hyperparameters.

**Bug Fixes**

-  WebUI: Fix the hyperparameter visualization page crashing when viewing single trial or PBT
   experiments, both of which are intentionally unsupported for hyperparameter visualizations.

Version 0.14.5
==============

**Release Date:** March 18, 2021

**New Features**

-  REST API: Add a REST API endpoint exposing historical cluster resource allocation. Currently,
   information about experiment workloads (training, checkpoints, and validations) is included.

-  Hyperparameter Search: Introduce a stopping-based variant of
   :ref:`topic-guides_hp-tuning-det_adaptive-asha` that will continue training trials by default
   unless stopped by the algorithm. Compared to the default promotions-based algorithm, the stopping
   variant will promote to higher rungs faster and does not require fault tolerance since it will
   not resume stopped trials.

-  PyTorch: Add an option to :class:`~determined.pytorch.LRScheduler` to accept a frequency option
   alongside batch and epoch step modes.

-  Kubernetes: Add support for priority scheduling, with gang-scheduling for distributed training,
   on Kubernetes.

**Improvements**

-  WebUI: Add a margin of comparison to hyperparameter visualizations to enable better grouping of
   trials with a similar but not identical number of batches processed.

**Bug Fixes**

-  Correct model code uploaded to checkpoints so it now matches the model code provided during
   experiment creation. Previously, it may have included additional files that had been bind-mounted
   with a ``container_path`` that was either relative or was a subdirectory of
   ``/run/determined/workdir``.

-  Fix an unauthorized access issue when attempting to use the Determined CLI within a notebook.

Version 0.14.3
==============

**Release Date:** March 4, 2021

**New Features**

-  Examples: Add the `Deformable DETR <https://openreview.net/forum?id=gZ9hCDWe6ke>`__ model for
   object detection in Determined. Check out `our example
   <https://github.com/determined-ai/determined-examples/tree/main/computer_vision/deformabledetr_coco_pytorch>`__.

-  Searcher: Support programmatic rejection of certain hyperparameters to further optimize your
   hyperparameter search.

-  WebUI: Add additional hyperparameter visualizations to multi-trial experiments. The new parallel
   coordinate, scatter plot, and heat map visualizations will allow you to better explore
   relationships between hyperparameters and model performance.

**Improvements**

-  WebUI: Use anchor tags instead of click event listeners across all table rows. This increases
   accessibility and improves keyboard navigation support.

**Bug Fixes**

-  Keras: Ensure that ``keras.utils.Sequence`` objects receive their ``on_epoch_end()`` calls after
   validation is completed.
-  WebUI: Fix the order of batches to be numeric instead of alphanumeric.

Version 0.14.2
==============

**Release Date:** February 17, 2021

**New Features**

-  Support CUDA 11. New Docker images are available for experiments and commands to support CUDA 11,
   as well as some updated versions of frameworks on CUDA 10.1. It is recommended to validate the
   performance of models when changing CUDA versions as some models can experience significant
   changes in training time, etc.

-  Support ``startup-hook.sh`` for notebooks and shells. This is the same mechanism supported by
   experiments.

**Improvements**

-  Improve local test mode for experiment creation, ``det experiment create``, to test with only a
   single batch.

-  Invoke ``python`` as ``python3`` rather than as ``python3.6``. This makes it possible to use
   custom images containing higher versions of Python with Determined (3.6 is still the minimum
   required version).

   If the desired ``python`` cannot be found as ``python3``, it is now possible to customize this
   invocation by setting the environment variable ``DET_PYTHON_EXECUTABLE=/path/to/python3``, for
   experiments, notebooks, and shells.

**Bug Fixes**

-  Kubernetes: Fix a bug that caused the Cluster page to not render when using a Kubernetes cluster.

Version 0.14.1
==============

**Release Date:** February 9, 2021

**Bug Fixes**

-  Trial: Fix a bug that prevented trial logs created before 0.13.8 from loading correctly.

Version 0.14.0
==============

**Release Date:** February 4, 2021

**New Features**

-  Add resource pools, which allows for different types of tasks to be scheduled onto different
   types of agents.
-  ``det-deploy`` will now create clusters with two resource pools, one that uses GPU instances and
   one that uses CPU instances for tasks that only require CPUs.
-  WebUI: Revamp cluster page with information about configured resource pools.

**Removed Features**

-  Trial API: Remove the old PyTorch APIs, including:

   -  the ``build_model``, ``optimizer``, and ``create_lr_scheduler`` methods in
      :class:`~determined.pytorch.PyTorchTrial`;

   -  the callback ``on_before_optimizer_step``;

   -  the field ``optimizations.mixed_precision`` in the experiment configuration;

   -  the ``model`` arguments to :meth:`~determined.pytorch.PyTorchTrial.train_batch`,
      :meth:`~determined.pytorch.PyTorchTrial.evaluate_batch`, and
      :meth:`~determined.pytorch.PyTorchTrial.evaluate_full_dataset`.

   Model code that uses these APIs will no longer run in Determined 0.14.0 or later. However, model
   checkpoints produced by old experiments that used these APIs will still be supported.

**Improvements**

-  **Breaking Change** REST API: The trial and checkpoint API endpoints can now return non-scalar
   metric values, which are represented as JSON objects or `protobuf structs
   <https://protobuf.dev/reference/protobuf/google.protobuf/#struct>`__.

-  Documentation: Add a topic guide on :ref:`debugging models <model-debug>`. The new guide will
   walk you step-by-step through solving problems with a model in Determined, with a focus on
   testing features incrementally until the model is fully working. It may also be useful when
   porting new models to Determined.

-  Documentation: Add a topic guide on :ref:`commands and shells <commands-and-shells>`. It
   describes how to use Determined's support for managing GPU-powered batch commands and interactive
   shells.

-  REST API: Improve the performance of the experiments API.

**Bug Fixes**

-  Database: Migrate ``public.trial_logs.id`` to be an ``int8`` in Postgres, instead of an ``int4``.
   This avoids issues for customers with extremely large amounts of trial logs. **Note**: This
   migration will be more time-consuming than usual for deployments with large amounts of trial
   logs.

-  REST API: Fix an issue where requesting checkpoint or trial details of a trial that had
   non-scalar metric values associated with it would fail.

-  Trial: Fix an issue where the trial was not deallocating resources when it failed to write to the
   DB.

-  WebUI: Show better messaging for different learning curve edge cases.

-  WebUI: Fix sorting on the experiment trials table within the experiment detail page.

-  WebUI: Fix issue of incorrect trial log order when viewing oldest logs first.

-  WebUI: Update Cancel confirm button label to show `Confirm` to avoid double `Cancel` buttons.

-  WebUI: Improve the sorting behavior for numeric table columns.

**************
 Version 0.13
**************

Version 0.13.13
===============

**Release Date:** January 25, 2021

**New Features**

-  Update experiment details pages to include a learning curve visualization. This will enable a
   comparison of hyperparameter performance among many different trials within an experiment.
-  Support Elasticsearch as an alternative backend for logging.

**Improvements**

-  **Breaking Change:** REST API: Update trial logs API to return string IDs.

-  WebUI: Enable filtering of trial logs by agent, container, rank, log level, and timestamp.

-  WebUI: Improve section contrast on all pages.

-  Deployment: Add the command ``det-deploy aws list``, which shows all the CloudFormation stacks
   that are managed by ``det-deploy aws`` (using the tag ``managed-by: determined``). This only
   applies to new deployments since this version, not previous deployments.

-  Update examples to use the new PyTorch APIs.

**Deprecated Features**

-  The old PyTorch API was deprecated in 0.12.13 and will be removed in the next release. See the
   PyTorch migration guide for details on updating your PyTorch model code to use the new API.

Version 0.13.12
===============

**Release Date:** January 11, 2021

**Bug Fixes**

-  WebUI: Fix the Okta sign-in workflow.
-  WebUI: Fix an issue with unexpected hyperparameter types in experiment configuration.
-  WebUI: Fix trial metric workload duration reporting in the trial detail page.

Version 0.13.11
===============

**Release Date:** January 6, 2021

**Improvements**

-  Trials: Add experimental support for custom metric reducers with PyTorchTrial. This enables
   calculating advanced metrics like F1 score or mean IOU; returning multiple metrics from a single
   reducer is also supported. See
   :meth:`determined.pytorch.PyTorchExperimentalContext.wrap_reducer()` for detailed documentation
   and code snippets.

   See ``determined/examples/features/legacy/custom_reducers_mnist_pytorch`` for a complete example
   of how to use custom reducers. The example emits a per-class F1 score using the new custom
   reducer API.

-  Trials: Support more than 1 backward pass per optimizer step for distributed training in
   PyTorchTrial.

-  Logging: Allow the trial logging backend to be configured in Kubernetes-based deployments of
   Determined.

-  Agents: Add support for labels when starting agents with ``det-deploy``.

**Bug Fixes**

-  WebUI: Update the Trial Information Table to be usable on mobile devices.
-  HP Search: Fix a bug where ``adaptive_asha`` could run with more maximum concurrent trials than
   intended.
-  Scheduling: Fix a bug where command priority was not respected.

Version 0.13.10
===============

**Release Date:** December 10, 2020

**New Features**

-  WebUI: Add support for mobile and tablet devices. Check your experiment results on the go!
-  Scheduler: Update the priority scheduler to support specifying priorities and preemption.

**Improvements**

-  Improve the scheduling and scaling behavior of CPU tasks, and allow the maximum number of CPU
   tasks per agent to be configured via the :ref:`cluster-configuration`.

-  Add custom tagging support to AWS dynamic agents. Thank you to ``sean-adler`` for contributing
   this improvement!

-  Support ``validation_steps`` in ``TFKerasTrial``'s ``context.configure_fit()``.
   ``validation_steps`` means the same thing in Determined as it does in ``model.fit()``, and has
   the same limitation (in that it only applies when ``validation_data`` is of type
   ``tf.data.Dataset``).

-  Kubernetes: Support a default user password for Kubernetes deployments. This affects the
   ``admin`` and ``determined`` default user accounts.

-  Kubernetes: Release version ``0.3.1`` of the Determined Helm chart.

**Bug Fixes**

-  Fix a bug in ``--local --test`` mode where all GPUs were being passed to the training loop
   despite the distributed training code paths being disabled.

-  Fix a bug causing `active` trials that have failed to not be restored properly on a master
   restart when ``max_restarts`` is greater than ``0``.

-  Allow configurations with a ``.`` character in the keys for map fields in the
   :ref:`master-config-reference` (e.g. ``task_container_defaults.cpu_pod_spec.metadata.labels``).

-  Fix a bug where restoring a large number of experiments after a failure could lead to deadlock.

-  Fix an issue where templates with user-specified bind mounts would merge incorrectly. Thank you
   to ``zjorgensenbits`` for `reporting this issue
   <https://github.com/determined-ai/determined/issues/1660>`__!

**Deprecated Features**

-  The previous version of the priority scheduler is now deprecated. It will remain available as the
   ``round_robin`` scheduler for a limited period of time.

Version 0.13.9
==============

**Release Date:** November 20, 2020

**Improvements**

-  Commands: Support configuring ``shmSize`` for commands (e.g., notebooks, shells, TensorBoards) in
   :ref:`command configurations <command-notebook-configuration>`.

**Bug Fixes**

-  API: Fix a bug that caused the WebUI's log viewer to fail to render previous pages of trial logs.
-  WebUI: Fix a bug in opening TensorBoards from the experiment list page via batch selection.

Version 0.13.8
==============

**Release Date:** November 17, 2020

**New Features**

-  API: Add support for models that subclass ``tf.keras.Model`` when using the Determined
   TFKerasTrial API. This is a new feature that became available starting in TensorFlow 2.2,
   allowing user to further customize their training process.

-  Deployment: When using the ``simple`` deployment type with ``det-deploy aws``, you can now use
   the ``--agent-subnet-id`` flag to specify which existing subnet to launch agents in. As each
   subnet is associated with a single availability zone, this allows users to explicitly choose an
   availability zone that has GPU instances (there is no public information about which availability
   zones have GPU instances so trial and error is the suggested approach).

-  Logs: Support filtering trial logs by individual fields in the CLI. Log entries for trials can
   now be filtered by container ID, agent ID, log level, and other fields.

-  Security: Allow the master to use a TLS certificate that is valid for a different name than the
   agents use to connect to it. This ability is useful in situations where the master is accessed
   using multiple different addresses (e.g., private and public IP addresses of a cloud instance).
   The agent now accepts a ``--security-tls-master-cert-name`` option to override the expected name
   in the master's TLS certificate. The CLI uses the ``DET_MASTER_CERT_NAME`` environment variable
   for the same purpose."

**Improvements**

-  **Breaking Change:** API: Perform salting and hashing on server-side for the password change
   endpoint. This makes this endpoint consistent with the new login endpoint described at
   https://docs.determined.ai/latest/rest-api/ .

-  **Breaking Change:** Logging: Start using Fluent Bit for handling trial logs internally. The
   agent machines now need to have access to the ``fluent/fluent-bit:1.6`` Docker image. If the
   Determined agent machines are able to connect to Docker Hub, they will pull it automatically and
   no changes are required; if not, the image must be manually made available beforehand. The
   Determined agent accepts a ``--fluent-logging-image`` option to specify an alternate name for the
   image. This change is part of an effort to improve the handling of trial logs by increasing
   scalability and allowing more options for log storage.

-  Agent: Support configurable slot types for agents. Previously, Determined only supported
   auto-detecting the slot type for agents. If Determined did not detect any GPUs, the agents would
   fall back to mapping one slot to all the CPUs. With this change, this behavior can be configured
   to one of ``auto``, ``gpu``, and ``none`` in the field ``slot_type`` of the agent configuration
   ``agent.yaml``. Dynamic agents having GPUs will be configured to ``gpu`` while those agents
   having no GPUs will be configured to ``none``. For static agents this field defaults to ``auto``.

-  API: Add ``self.context.wrap_optimizer()`` to the Determined TFKerasTrial API.

-  API: Add tf.keras DCGAN example that subclasses ``tf.keras.Model``.

-  API: Add ``self.context.configure_fit()`` to the Determined TFKerasTrial API. Many parameters
   which would be passed to ``model.fit()``, such as ``class_weight``, ``verbose``, or ``workers``,
   can now be passed to ``configure_fit()`` and will be honored by ``TFKerasTrial``.

-  Kubernetes: Add option to configure the service type of the Determined deployed database in the
   Determined Helm chart. This is useful if your cluster does not support ClusterIP, which is the
   service type that is used by default.

-  WebUI: Make the page/tab title more descriptive.

-  WebUI: Add navigation sidebar, breadcrumb, and back buttons to log view pages.

-  WebUI: Update the trial and master log buttons to open in the same page by default, with the
   option to open in a new tab.

-  WebUI: Update trial details URL to include the experiment id.

**Bug Fixes**

-  API: Fix support for Keras Callbacks.

   -  Previously, stateful Keras Callbacks (``EarlyStopping`` and ``ReduceLROnPlateau``) did not
      work in Determined across pause/activate boundaries. We have introduced Determined-friendly
      implementations, :class:`determined.keras.callbacks.EarlyStopping` and
      :class:`determined.keras.callbacks.ReduceLROnPlateau`, which address this shortcoming.
      User-defined callbacks may subclass :class:`determined.keras.callbacks.Callback` (and define
      ``get_state`` and ``load_state`` methods) to also benefit from this and other new features.

   -  Previously, Keras Callbacks which relied on ``on_epoch_end`` in Determined would see their
      ``on_epoch_end`` called every ``scheduling_unit`` batches by default. Now, ``on_epoch_end``
      will be reliably called at the end of each epoch, as defined by the ``records_per_epoch``
      setting in the experiment config. As before, ``on_epoch_end`` will not contain validation
      metrics, as the validation data is not always fresh at epoch boundaries. Therefore, the
      Determined implementations of :class:`~determined.keras.callbacks.EarlyStopping` and
      :class:`~determined.keras.callbacks.ReduceLROnPlateau` are both based on ``on_test_end``,
      which can be tuned using ``min_validation_period``.

-  API: Fix issue that occasionally made TFKerasTrial hang for multi-GPU training during
   ``COMPUTE_VALIDATION_STEP``.

-  Kubernetes: Gracefully handle cases where the Kubernetes API server responds with unexpected
   object types.

-  Scheduler: Fix not being able to find resource pools for experiments.

-  Scheduler: Fix not being able to disable slots.

-  WebUI: Prevent navigation item tooltips from showing up when hovering outside of the navigation
   bar.

-  WebUI: Fix an issue where the experiment archive action button was out of sync.

-  WebUI: Fix experiment actions to not display a loading spinner.

**Deprecated Features**

-  API: Deprecate the name ``det.keras.TFKerasTensorBoard`` in favor of
   ``det.keras.callbacks.TensorBoard``. The old name will be removed eventually, and user code
   should be updated accordingly.

-  API: Deprecated the old ``det.keras.SequenceAdapter``. ``SequenceAdapter`` will be removed in a
   future version. Users should use ``self.context.configure_fit()`` instead, which is both more
   capable and more similar to the normal ``tf.keras`` APIs.

Version 0.13.7
==============

**Release Date:** October 29, 2020

**New Features**

-  Add support for running workloads on spot instances on AWS. Spot instances can be up to 70%
   cheaper than on-demand instances. If a spot instance is terminated, Determined's built-in fault
   tolerance means that model training will continue on a different agent automatically. Spot
   instances can be enabled by setting ``spot: true`` in the :ref:`cluster-configuration`.

-  Support `MMDetection <https://github.com/open-mmlab/mmdetection>`__, a popular library for object
   detection, in Determined. MMDetection allows users to easily train state-of-the-art object
   detection models; with Determined, users can take things one step further with cutting-edge
   distributed training and hyperparameter tuning to further boost performance. See the `Determined
   implementation of MMDetection
   <https://github.com/determined-ai/determined-examples/tree/main/model_hub/mmdetection>`__.

-  WebUI: Allow the experiments list page to be filtered by labels. Selecting more than one label
   will filter experiments by the intersection of the selected labels.

**Deprecated Features**

-  Deprecate the simple and advanced adaptive hyperparameter search algorithms. They will be removed
   in a future release. Both algorithms have been replaced with
   :ref:`topic-guides_hp-tuning-det_adaptive-asha`, which has state-of-the-art performance, as well
   as better scalability and resource-efficiency.

**Improvements**

-  Documentation: Add a guide for :ref:`setup-eks-cluster`.

-  Master: Support a minimum instance count for dynamic agents. The master will attempt to scale the
   cluster to at least the configured value at all times. This is configurable via
   ``provisioner.min_instances`` in the :ref:`cluster-configuration`. This will increase
   responsiveness to workload demand because agent(s) will be ready even when the cluster is idle.

-  Kubernetes: Improve the performance of the ``/agents`` endpoint for Kubernetes deployments. This
   will improve the performance of the cluster page in the WebUI, as well as when using ``det slot
   list`` and ``det task list`` via the CLI.

-  Kubernetes: Release version ``0.3.0`` of the Determined Helm chart.

-  WebUI: Improve metric selection on the trial detail page. This should improve filtering for
   trials with many metrics.

-  WebUI: Use scientific notation when appropriate for floating point metric values.

-  WebUI: Show both experiment and trial TensorBoard sources when applicable.

**Bug Fixes**

-  WebUI: Fix an issue where TensorBoard sources did not display properly for TensorBoards started
   via the CLI.
-  WebUI: Fix an issue with rendering boolean hyperparameters in the WebUI.
-  CLI: Fix an issue where trial IDs were occasionally not displayed when running ``det task list``
   or ``det slot list`` in the CLI.
-  Master: Fix the default value for the ``fit`` field if the ``scheduler`` is set in the
   :ref:`cluster-configuration`.

Version 0.13.6
==============

**Release Date:** October 14, 2020

**Improvements**

-  Agent: The ``boot_disk_source_image`` field for GCP dynamic agents and ``image_id`` field for AWS
   dynamic agents are now optional. If omitted, the default value is the Determined agent image that
   matches the Determined master being used.

-  Documentation: Ship Swagger UI with Determined documentation. The ``/swagger-ui`` endpoint has
   been renamed to ``/docs/rest-api``.

-  Documentation: Add a :ref:`guide on configuring TLS <tls>` in Determined.

-  Kubernetes: Add support for configuring memory and CPU requirements for the Determined database
   when installing via the Determined Helm chart.

-  Kubernetes: Add support for configuring the `storageClass
   <https://kubernetes.io/docs/concepts/storage/storage-classes/>`__ that is used when deploying a
   database using the Determined Helm chart.

**Bug Fixes**

-  Harness: Do not require the master to present a full TLS certificate chain when the certificate
   is signed by a well-known Certificate Authority.
-  Harness: Fix a bug which affected ``TFKerasTrial`` using TensorFlow 2 with
   ``gradient_aggregation`` > 1.
-  Master: Fix a bug where the master instance would fail if an experiment could not be read from
   the database.
-  WebUI: Preserve the colors used for multiple metrics on the metric chart.
-  WebUI: Fix the ability to cancel a batch of experiments.
-  WebUI: Fix a bug which caused the Experiment Details page to not render when the latest
   validation metric is not available.

Version 0.13.5
==============

**Release Date:** September 30, 2020

**Improvements**

-  Security: Use one TCP port for all incoming connections to the master and use TLS for all
   connections if configured.

   -  **Breaking Change:** The ``http_port`` and ``https_port`` options in the master configuration
      have been replaced by the single ``port`` option. The ``security.http`` option is no longer
      accepted; the master can no longer be configured to listen over HTTP and HTTPS simultaneously.

-  Security: Support configuring TLS encryption when deploying Determined on Kubernetes.

-  Agent: Increase default max agent starting and idle timeouts to 20 minutes and increase max
   disconnected period from 5 to 10 minutes.

-  Deployment: Add support for ``det-deploy aws`` in the following new regions: ``ap-northeast-1``,
   ``eu-central-1``, ``eu-west-1``, ``us-east-2``.

-  Docker: Publish new Docker task containers that upgrade TensorFlow versions from 1.15.0 to
   1.15.4, and 2.2.0 to 2.2.1.

-  Documentation: Add extra documentation and reorganize examples by use case.

-  Documentation: Add a ``tf.layers-in-Estimator`` example.

-  Kubernetes: Add support for users to specify ``initContainers`` and ``containers`` as part of
   their custom pod specs.

-  Kubernetes: Publish version 0.2.0 of the Determined Helm chart.

-  Native API: Deprecate Native API. Removed related examples and docs.

-  Trials: Remove support for ``TensorpackTrial``.

-  WebUI: Improve polling behavior for experiment and trial details pages to avoid hanging
   indefinitely for very large experiments/trials.

**Bug Fixes**

-  Trials: Fix a bug where if only a subset of workers on a machine executed the
   ``on_trial_close()`` ``EstimatorTrial`` callback, the container would terminate as soon as one
   worker exited.

-  Trials: Fix a bug where ``det e create --test`` would succeed when there were checkpointing
   failures.

-  WebUI: Fix the issue of multiple selected rows dissappearing after a successful table batch
   action.

-  WebUI: Remove unused TensorBoard sources column from the task list page.

-  WebUI: Fix rendering metrics with the same name on the metric chart.

-  WebUI: Make several fixes to improve select appearance and user experience.

-  WebUI: Fix the issue of agent and cluster info not loading on slow connections.

-  WebUI: Fix the issue where the chart in the Experiment page does not have the metric name in the
   legend.

Version 0.13.4
==============

**Release Date:** September 16, 2020

**Improvements**

-  Support configuring default values for the task image, Docker pull policy, and Docker registry
   credentials via the :ref:`master-config-reference` and the :ref:`helm-config-reference`. In
   previous versions of Determined, these values had to be specified on a per-task basis (e.g., in
   the experiment configuration). Per-task configuration is still supported and will overwrite the
   default value (if any).

-  Add connection checks for dynamic agents. A dynamically provisioned agent will be terminated if
   it is not actively connected to the master for at least five minutes.

-  Emit a warning if ``DistributeConfig`` is specified for an ``Estimator``. Configuring an
   ``Estimator`` via ``tf.distribute.Strategy`` can conflict with how Determined performs
   distributed training. With this change, Determined will attempt to catch this problem and surface
   an error message in the experiment logs. An ``Estimator`` can still be configured with an empty
   ``DistributeConfig`` without issue.

-  Remove support for ``dataflow_to_tf_dataset`` in :class:`~determined.estimator.EstimatorTrial`.
   Dataflows should be wrapped using ``wrap_dataset(shard=False)`` instead.

-  WebUI: Add middle mouse button click detection on tables to open in a new tab/page.

-  WebUI: Improve the trial detail metrics view.

   -  Support metrics with non-numeric values.
   -  Default to showing only the searcher metric on initial page load.
   -  Add search capability to the metric select filter. This should improve the experience when
      there are many metrics.
   -  Add support for displaying multiple metrics on the metric chart.

-  WebUI: Move TensorBoard sources from a table column into a separate modal.

-  WebUI: Optimize loading of active TensorBoards and notebooks.

**Bug Fixes**

-  Improve handling of certain corner cases where distributed training jobs could hang indefinitely.
-  Fix an issue where detecting GPU availability in TensorFlow code would cause ``EstimatorTrial``
   models to OOM.
-  Fix an issue where accessing logs could create a memory leak.
-  Fix an issue that prevents resuming from checkpoints that contain a large number of files.
-  WebUI: Fix an issue where table page sizes were not saved between page loads.
-  WebUI: Fix an issue where opening a TensorBoard on an experiment would not direct the user to an
   already running TensorBoard, but instead create a new one.
-  WebUI: Fix an issue where batch actions on the experiments table would cause rows to disappear.

**Known Issues**

-  WebUI: In the trial detail metrics view, experiments that have both a training metric and a
   validation metric of the same name will not be displayed correctly on the metrics chart.

Version 0.13.3
==============

**Release Date:** September 8, 2020

**Bug Fixes**

-  Deployment: Fix a bug where ``det-deploy local cluster-up`` was failing.
-  WebUI: Fix a bug where experiment labels were not displayed on the experiment list page.
-  WebUI: Fix a bug with decoding API responses because of unexpected non-numeric metric values.

Version 0.13.2
==============

**Release Date:** September 3, 2020

**New Features**

-  Support deploying Determined on `Kubernetes <https://kubernetes.io/>`__.

   -  Determined workloads run as a collection of pods, which allows standard Kubernetes tools for
      logging, metrics, and tracing to be used. Determined is compatible with Kubernetes >= 1.15,
      including managed Kubernetes services such as Google Kubernetes Engine (GKE) and AWS Elastic
      Kubernetes Service (EKS).

   -  When using Determined with Kubernetes, we currently do not support fair-share scheduling,
      priority scheduling, per-experiment weights, or gang-scheduling for distributed training
      experiments; workloads will be scheduled according the behavior of the default Kubernetes
      scheduler.

   -  Users can configure the behavior of the pods that are launched for Determined workloads by
      specifying a :ref:`custom pod spec <custom-pod-specs>`. A default pod spec can be configured
      when installing Kubernetes, but a custom pod spec can also be specified on a per-task basis
      (e.g., via the :ref:`environment.pod_spec <exp-environment-pod-spec>` field in the experiment
      configuration file).

-  Support running multiple distributed training jobs on a single agent.

   -  In previous versions of Determined, a distributed training job could only be scheduled on an
      agent if it was configured to use all of the GPUs on that agent. In this release, that
      restriction has been lifted: for example, an agent with 8 GPUs can now be used to run two
      4-GPU distributed training jobs. This feature is particularly useful as a way to improve
      utilization and fair resource allocation for smaller clusters.

**Improvements**

-  WebUI: Update primary navigation. The primary navigation is all to one side, and is now
   collapsible to maximize content space.

-  WebUI: Trial details improvements:

   -  Update metrics selector to show the number of metrics selected to improve readability.
   -  Add the "Has Checkpoint or Validation" filter.
   -  Persist the "Has Checkpoint or Validation" filter setting across all trials, and persist the
      "Metrics" filter on trials of the same experiment.

-  WebUI: Improve table pagination behavior. This will improve performance on Determined instances
   with many experiments.

-  WebUI: Persist the sort order and sort column for the experiments, tasks, and trials tables to
   local storage.

-  WebUI: Improve the default axes' ranges for metrics charts. Also, update the range as new data
   points arrive.

-  Add a warning when the PyTorch LR scheduler incorrectly uses an unwrapped optimizer. When using
   PyTorch with Determined, LR schedulers should be constructed using an optimizer that has been
   wrapped via the :meth:`~determined.pytorch.PyTorchTrialContext.wrap_optimizer` method.

-  Add a reminder to remove ``sys.exit()`` if ``SystemExit`` exception is caught.

**Bug Fixes**

-  WebUI: Fix an issue where the recent task list did not apply the limit filter properly.
-  Fix Keras and Estimator wrapping functions not returning the original objects when exporting
   checkpoints.
-  Fix progress reporting for ``adaptive_asha`` searches that contain failed trials.
-  Fix an issue that was causing OOM errors for some distributed ``EstimatorTrial`` experiments.

Version 0.13.1
==============

**Release Date:** August 31, 2020

**Bug Fixes**

-  Database migration: Fix a bug with a database migration in Determined version 0.13.0 which caused
   it to run slow and backfill incorrect values. Users on Determined versions 0.12.13 or earlier are
   recommended to upgrade to version 0.13.1. Users already on version 0.13.0 should upgrade to
   version 0.13.1 as usual.

-  TensorBoard: Fix a bug that prevents TensorBoards from experiments with old experiment
   configuration versions from being loaded.

-  WebUI: Fix an API response decoding issue on React where a null checkpoint resource was unhandled
   and could prevent trial detail page from rendering.

-  WebUI: Fix an issue where terminated TensorBoard and notebook tasks were rendered as openable.

Version 0.13.0
==============

**Release Date:** August 20, 2020

This release of Determined introduces several significant new features and modifications to existing
features. When upgrading from a prior release of Determined, users should pay particular attention
to the following changes:

-  The concept of "steps" has been removed from the CLI, WebUI, APIs, and configuration files.
   Before upgrading, **terminate all active and paused experiments** (e.g., via ``det experiment
   cancel`` or ``det experiment kill``). The format of the experiment config file has changed --
   configuration files that worked with previous versions of Determined will need to be updated to
   work with Determined >= 0.13.0.

-  The WebUI has been partially rewritten, moving several components that were implemented in Elm to
   now being written in React and TypeScript. As part of this change, many improvements to the
   performance, appearance, and usability of the WebUI have been made. For more details, see the
   list of changes below. Please notify the Determined team of any regressions in functionality.

-  The usability of the ``det shell`` feature has been significantly enhanced. As part of this
   change, the way in which arguments to ``det shell`` are parsed has changed; see details below.

*We recommend taking a backup of the database before upgrading Determined.*

**New Features**

-  Allow trial containers to connect to the master using TLS.

-  Allow agent's TLS verification to skip verification or use a custom certificate for the master.

-  For :class:`~determined.keras.TFKerasTrial` and :class:`~determined.estimator.EstimatorTrial`,
   add support for disabling automatic sharding of the training dataset when doing distributed
   training. When wrapping a dataset via ``context.wrap_dataset``, users can now pass
   ``shard_dataset=False``. If this is done, users are responsible for splitting their dataset in
   such a manner that every GPU (rank) sees unique data.

**Improvements**

-  **Remove Steps from the UX:** Remove the concept of a "step" from the CLI, WebUI, and
   configuration files. Add new configuration settings to allow settings previously in terms of
   steps to be configured instead in terms of records, batches or epochs..

   -  Many configuration settings can now be set in terms of records, batches or epochs. For
      example, a single searcher can be configured to run for 100 records by setting ``max_length:
      {records: 100}``, 100 batches by setting ``max_length: {batches: 100}``, or 100 epochs by
      setting ``records_per_epoch`` at the root of the config and ``max_length: {epochs: 100}``.

   -  A new configuration setting, ``records_per_epoch``, is added that must be specified when any
      quantity is configured in terms of epochs.

   -  **Breaking Change:** For single, random and grid searchers ``searcher.max_steps`` has been
      replaced by ``searcher.max_length``

   -  **Breaking Change:** For ASHA based searchers, ``searcher.target_trial_steps`` and
      ``searcher.step_budget`` has been replaced by ``searcher.max_length`` and ``searcher.budget``,
      respectively.

   -  **Breaking Change:** For PBT, ``searcher.steps_per_round`` has been replaced by
      ``searcher.length_per_round``.

   -  **Breaking Change:** For all experiments, the names for ``min_validation_period`` and
      ``min_checkpoint_period`` are unchanged but they are now configured in terms of records,
      batches or epochs.

-  **Shell Mode Improvements:** Determined supports launching GPU-attached terminal sessions via
   ``det shell``. This release includes several changes to improve the usability of this feature,
   including:

   -  The ``determined`` and ``determined-cli`` Python packages are now automatically installed
      inside containers launched by ``det shell``. Any user-defined environment variables for the
      task image will be passed into the ssh sessions opened via ``det shell start`` or ``det shell
      open``.

   -  ``det shell`` should now work correctly in "host" networking mode.

   -  ``det shell`` should now work correctly with dynamic agents and in cloud environments.

   -  **Breaking Change:** Change how additional arguments to ``ssh`` are passed through ``det shell
      start`` and ``det shell open``. Previously they were passed as a single string, like ``det
      shell open SHELL_ID --ssh-opt '-X -Y -o SomeSetting="some string"'``, but now the
      ``--ssh-opt`` has been removed and all extra positional arguments are passed through without
      requiring double-layers of quoting, like ``det shell open SHELL_ID -- -X -Y -o
      SomeSetting="some string"`` (note the use of ``--`` to indicate all following arguments are
      positional arguments).

-  **WebUI changes**

   -  Tasks List: ``/det/tasks``

      -  Consolidate notebooks, tensorboards, shells, commands into single list page.
      -  Add type filter to control which task types to display. By default all task types are shown
         when none of the types are selected.
      -  Add type column with iconography to train users to familiarize task types with visual
         indicators.
      -  Convert State filter from multi-select to single-select.
      -  Convert actions from expanded buttons to overflow menu (triple vertical dots).
      -  Move notebook launch buttons to task list from notebook list page.
      -  Add pagination support that auto turns on when entries extend beyond 10 entries.
      -  Add list of TensorBoard sources in a table Source column.

   -  Experiment List: ``/det/experiments``

      -  State filter converted from multi-select to single-select.
      -  Convert actions from expanded buttons to overflow menu (triple vertical dots).
      -  Batch operation logic change to available if the action can be applied to any of the
         selected experiments
      -  Add pagination support that auto turns on when entries extend beyond 10 entries.

   -  Experiment Detail: ``/det/experiments/<id>``

      -  Implement charting with Plotly with zooming capability.
      -  Trial table paginates on the WebUI side in preparation for API pagination in the near
         future.
      -  Convert steps to batches in trials table and metric chart.
      -  Update continue trial flow to use batches, epochs or records.
      -  Use Monaco editor for the experiment config with YAML syntax highlighting.
      -  Add links to source for Checkpoint modal view, allowing users to navigate to the
         corresponding experiment or trial for the checkpoint.

   -  Trial Detail: ``/det/trials/<id>``

      -  Add trial information table.
      -  Add trial metrics chart.
      -  Implement charting with Plotly with zooming capability.
      -  Trial info table paginates on the WebUI side in preparation for API pagination in the near
         future.
      -  Add support for batches, records and epochs for experiment config.
      -  Convert metric chart to show batches.
      -  Convert steps table to batches table.

   -  Master Logs: ``/det/logs``, Trial Logs: ``/det/trials/<id>/logs``, Task Logs:
      ``/det/<tasktype>/<id>/logs``

      -  Limit logs to 1000 lines for initial load and load an additional 1000 for each subsequent
         fetch of older logs.
      -  Use new log viewer optimized for efficient rendering.
      -  Introduce log line numbers.
      -  Add ANSI color support.
      -  Add error, warning, and debug visual icons and colors.
      -  Add tailing button to enable tailing log behavior.
      -  Add scroll to top button to load older logs out
      -  Fix back and forth scrolling behavior on log viewer.

   -  Cluster: ``/det/cluster``

      -  Separate out GPU from CPU resources.
      -  Show resource availability and resource count (per type).
      -  Render each resource as a donut chart.

   -  Navigation

      -  Update sidebar navigation for new task and experiment list pages.
      -  Add link to new swagger API documentation.
      -  Hide pagination controls for tables with less than 10 entries.

**Bug Fixes**

-  Configuration: Do not load the entire experiment configuration when trying to check if an
   experiment is valid to be archived or unarchived.

-  Configuration: Improve the master to validation hyperparameter configurations when experiments
   are submitted. Currently, the master checks whether ``global_batch_size`` has been specified and
   if it is numeric.

-  Logs: Fix issue of not detecting newlines in the log messages, particularly Kubernetes log
   messages.

-  Logs: Add intermediate step to trial log download to alert user that the CLI is the recommended
   action, especially for large logs.

-  Searchers: Fix a bug in the SHA searcher caused by the promotion of already-exited trials.

-  Security: Apply user authentication to streaming endpoints.

-  Tasks: Allow the master certificate file to be readable even for a non-root task.

-  TensorBoard: Fix issue affecting TensorBoards on AWS in us-east-1 region.

-  TensorBoard: Recursively search for tfevents files in subdirectories, not just the top level log
   directory.

-  WebUI: Fix scrolling issue that occurs when older logs are loaded, the tailing behavior is
   enabled, and the view is scrolled up.

-  WebUI: Fix colors used for different states in the cluster resources chart.

-  WebUI: Correct the numbers in the ``Batches`` column on the experiment list page.

-  WebUI: Fix cluster and dashboard reporting for disabled slots.

-  WebUI: Fix issue of archive/unarchive not showing up properly under the task actions.

**************
 Version 0.12
**************

Version 0.12.13
===============

**Release Date:** August 6, 2020

**New Features**

-  **Model Registry:** Determined now includes a built-in model registry, which makes it easy to
   organize trained models by providing versioning and labeling tools.

-  **New PyTorch API:** Add a new version of the PyTorch API that is more flexible and supports deep
   learning experiments that use multiple models, optimizers, and LR schedulers. The old API is
   still supported but is now deprecated and will be removed in a future release. See the `PyTorch
   migration guide
   <https://docs.determined.ai/0.12.13/reference/api/pytorch.html#migration-from-deprecated-interface>`_
   for details on updating your PyTorch model code. *Deprecated methods will be supported until at
   least the next minor release.*

   -  The new API supports PyTorch code that uses multiple models, optimizers, and LR schedulers. In
      your trial class, you should instantiate those objects and wrap them with
      :meth:`~determined.pytorch.PyTorchTrialContext.wrap_model`,
      :meth:`~determined.pytorch.PyTorchTrialContext.wrap_optimizer`, and
      :meth:`~determined.pytorch.PyTorchTrialContext.wrap_lr_scheduler` in the constructor of your
      PyTorch trial class. The previous API methods ``build_model``, ``optimizer``, and
      ``create_lr_scheduler`` in :class:`~determined.pytorch.PyTorchTrial` are now deprecated.

   -  Support customizing forward and backward passes in
      :meth:`~determined.pytorch.PyTorchTrial.train_batch`. Gradient clipping should now be done by
      passing a function to the ``clip_grads`` argument of
      :meth:`~determined.pytorch.PyTorchTrialContext.step_optimizer`. The callback
      ``on_before_optimizer_step`` is now deprecated.

   -  Configuring automatic mixed precision (AMP) in PyTorch should now be done by calling
      :meth:`~determined.pytorch.PyTorchTrialContext.configure_apex_amp` in the constructor of your
      PyTorch trial class. The ``optimizations.mixed_precision`` experiment configuration key is now
      deprecated.

   -  The ``model`` arguments to :meth:`~determined.pytorch.PyTorchTrial.train_batch`,
      :meth:`~determined.pytorch.PyTorchTrial.evaluate_batch`, and
      :meth:`~determined.pytorch.PyTorchTrial.evaluate_full_dataset` are now deprecated.

-  **More Efficient Hyperparameter Search:** This release introduces a new hyperparameter search
   method, ``adaptive_asha``. This is based on an asynchronous version of the ``adaptive``
   algorithm, and should enable large searches to find high-quality hyperparameter configurations
   more quickly.

**Improvements**

-  Allow proxy environment variables to be set in the agent config.
-  Preserve random state for PyTorch experiments when checkpointing and restoring.
-  Remove ``determined.pytorch.reset_parameters()``. This should have no effect except when using
   highly customized ``nn.Module`` implementations.
-  WebUI: Show total number of resources in the cluster resource charts.
-  Add support for NVIDIA T4 GPUs.
-  ``det-deploy``: Add support for ``g4`` instance types on AWS.
-  Upgrade NVIDIA drivers on the default AWS and GCP images from ``410.104`` to ``450.51.05``.

**Bug Fixes**

-  Fix an issue with the SHA searcher that could cause searches to stop making progress without
   finishing.
-  Fix an issue where ``$HOME`` was not properly set in notebooks running in nonroot containers.
-  Fix an issue where killed experiments had their state reset to the latest checkpoint.
-  Randomize the notebook listening port to avoid port binding issues in host mode.

Version 0.12.12
===============

**Release Date:** July 22, 2020

**Improvements**

-  Remove support for ``on_train_step_begin`` and ``on_train_step_end``, deprecate
   ``on_validation_step_end``, and introduce new callback ``on_validation_end`` with same
   functionality. Add helper methods ``is_epoch_start`` and ``is_epoch_end`` to PyTorch context.

-  Add a new API to support custom reducers in ``EstimatorTrial``.

-  CLI: Add the ``register_version`` command for registering a new version of a model.

-  CLI: Add a ``--head`` option when printing trial logs.

-  WebUI: Make it possible to launch TensorBoard from experiment dashboard cards.

**Bug Fixes**

-  Fix distributed training and Determined shell with non-root containers. The default task
   environments now include a user plugin to support running containers with arbitrary non-root
   users. Custom images based on the latest default task environments should also work.

-  Fix convergence issue for TF 2 multi-GPU models. Change default TF1 version from 1.14 to 1.15.

-  Fix issue affecting TensorFlow TensorBoard outputs.

-  Use local log line IDs for trial logs.

-  CLI: Improve the CLI's custom TLS certificate handling with non-self-signed certs.

-  WebUI: Fix a parsing problem with task start times.

-  WebUI: Fix log viewer timestamp copy/paste.

**Known Issues**

-  WebUI: Older trial logs are not loaded by scrolling to the top of the page.

Version 0.12.11
===============

**Release Date:** July 8, 2020

-  Add logging to console in test mode for the Native API when using
   :class:`determined.experimental.create`.

-  Improve reliability of saving checkpoints to GCS in the presence of transient network errors.

-  Add `an example
   <https://github.com/determined-ai/determined-examples/tree/main/computer_vision/unets_tf_keras>`__
   using TensorFlow's *Image Segmentation via UNet* tutorial.

-  WebUI: Improve trial log rendering performance.

-  WebUI: Fix an issue where cluster utilization was displayed incorrectly.

-  WebUI: Fix an issue where active experiments and commands would not appear on the dashboard.

-  WebUI: Fix an issue where having telemetry enabled with an invalid key would cause the WebUI to
   render incorrectly.

Version 0.12.10
===============

**Release Date:** June 26, 2020

**Improvements**

-  WebUI: Add a dedicated page for master logs at ``/det/logs``.
-  WebUI: Provide a Swagger UI for exploring the Determined REST API. This can be accessed via the
   API link on the WebUI.
-  WebUI: Default the Experiments view list length to 25 entries. More entries can be shown as
   needed.
-  WebUI: Improve detection of situations where the WebUI version doesn't match the master version
   as a result of browser caching.
-  CLI: Improve performance when retrieving trial logs.
-  CLI: Add the ``det user rename`` command for administrators to change the username of existing
   users.
-  Expand documentation on :ref:`use-trained-models` by including checkpoint metadata management.
-  Reorganize examples by splitting the Trial examples into separate folders.

**Bug Fixes**

-  Allow ``det-deploy local agent-up`` to work with remote masters.
-  Ensure network failures during checkpoint upload do not unrecoverably break the associated trial.
-  Ensure ``shared_fs`` checkpoint storage is usable for non-root containers for some ``host_path``
   values.
-  Fix a timeout issue that affected large (40+ machines) distributed experiments.
-  Ensure the CLI can make secure connections to the master.
-  Fix an issue that affected multi-GPU in ``PyTorchTrial`` with mixed precision enabled.
-  Add a timeout to trial containers to ensure they are terminated promptly.

Version 0.12.9
==============

**Release Date:** June 16, 2020

-  Retry ``ConnectionError`` and ``ProtocolError`` types for uploads to Google Cloud Storage.
-  Fix a bug where the CLI was unable to make secure websocket connections to the master.
-  Add the ``det user rename`` CLI command for admins to change the username of existing users.

Version 0.12.7
==============

**Release Date:** June 11, 2020

-  **Breaking Change:** Gradient clipping for PyTorchTrial should now be specified via
   :class:`determined.pytorch.PyTorchCallback` via the ``on_before_optimizer_step()`` method instead
   of being specified via the experiment configuration. Determined provides two built-in callbacks
   for gradient clipping: :class:`determined.pytorch.ClipGradsL2Norm` and
   :class:`determined.pytorch.ClipGradsL2Value`.

-  Add a ``metadata`` field to checkpoints. Checkpoints can now have arbitrary key-value pairs
   associated with them. Metadata can be added, queried, and removed via the :class:`Python SDK
   <determined.experimental.Checkpoint>`.

-  Add support for Keras callbacks that stop training early, including the `official EarlyStopping
   callback <https://www.tensorflow.org/api_docs/python/tf/keras/callbacks/EarlyStopping>`__. When a
   stop is requested, Determined will finish the training (or validation) step we are in,
   checkpoint, and terminate the trial.

-  Add support for Estimator callbacks that stop training early, including the official
   `stop_if_no_decrease_hook
   <https://www.tensorflow.org/versions/r2.11/api_docs/python/tf/estimator/experimental/stop_if_no_decrease_hook>`__.
   When a stop is requested, Determined will finish the training (or validation) step we are in,
   checkpoint, and terminate the trial.

-  Add support for model code that stops training of a trial programmatically.

   -  We recommend using the official Keras callbacks or Estimator hooks if you are using those
      frameworks. For PyTorch, you can request that training be stopped by calling
      :meth:`~determined.TrialContext.set_stop_requested` from a PyTorch callback. When a stop is
      requested, Determined will finish the current training or validation step, checkpoint, and
      terminate the trial. Trials that are stopped early are considered to be "completed" (e.g., in
      the WebUI and CLI).

-  More robust error handling for hyperparameter searches where one of the trials in the search
   encounters a persistent error.

   -  Determined will automatically restart the execution of trials that fail within an experiment,
      up to ``max_restart`` failures. After this point, any trials that fail are marked as "errored"
      but the hyperparameter search itself is allowed to continue running. This is particularly
      useful when some parts of the hyperparameter space result in models that cannot be trained
      successfully (e.g., the search explores a range of batch sizes and some of those batch sizes
      cause GPU OOM errors). An experiment can complete successfully as long as at least one of the
      trials within it completes successfully.

-  Support multi-GPU training for TensorFlow 2 models that use ``IndexedSlices`` for model
   parameters.

-  ``NaN`` values in training and validation metrics are now treated as errors.

   -  This will result in restarting the trial from the most recently checkpoint if it has been
      restarted fewer than ``max_restarts`` times. Previously, ``NaN`` values were converted to the
      maximum floating point value.

-  Preserve the last used user name on the log-in page.

-  Add ``on_trial_close`` method to :class:`determined.estimator.RunHook`. Use this for post-trial
   cleanup.

-  Finalize gradient communication prior to applying gradient clipping in PyTorchTrial when
   perfoming multi-GPU training.

-  WebUI: Add pause, activate, and cancel actions to dashboard tasks.

-  Add a ``det-nobody`` user (with UID 65533) to default images. This provides an out-of-the-box
   option for running non-privileged containers with a working home directory.

Version 0.12.6
==============

**Release Date:** June 5, 2020

-  Add end of training callback to EstimatorTrial.

Version 0.12.5
==============

**Release Date:** May 27, 2020

-  **Breaking Change:** Alter command-line options for controlling test mode and local mode. Test
   experiments on the cluster were previously created with ``det e create --test-mode ...`` but now
   should be created with ``det e create --test ...``. Local testing is started with ``det e create
   --test --local ...``. Fully local training (meaning ``--local`` without ``--test``) is not yet
   supported.

-  Add support for TensorFlow 2.2.

-  Add support for post-checkpoint callbacks in :class:`~determined.pytorch.PyTorchTrial`.

-  Add support for checkpoint hooks in :class:`~determined.estimator.EstimatorTrial`.

-  Add support for TensorBoard backed by S3-compliant APIs that are not AWS S3.

-  Add generic callback support for PyTorch.

-  TensorBoards now shut down after 10 minutes if metrics are unavailable.

-  Update to NCCL 2.6.4 for distributed training.

-  Update minimum required task environment version to 0.4.0.

-  Fix Native API training one step rather than one batch when using TensorFlow Keras and Estimator.

-  CLI: Add support for producing CSV and JSON output to ``det slot list`` and ``det agent list``.

-  CLI: Include the number of containers on each agent in the output of ``det agent list``.

Enterprise:

-  Add support for using SCIM (System for Cross-domain Identity Management) to provision users.
-  Add support for using OAuth2 to secure Determined's SCIM integration.
-  Add support for users to sign-on through an external IdP with SAML.

Version 0.12.4
==============

**Release Date:** May 14, 2020

-  **Breaking Change:** Users are no longer automatically logged in as the "determined" user.

-  Support multi-slot notebooks. The number of slots per notebook cannot exceed the size of the
   largest available agent. The number of slots to use for a notebook task can be configured when
   the notebook is launched: ``det notebook start --config resources.slots=2``

-  Support fetching the configuration of a running master via the CLI (``det master config``).

-  Authentication sessions now expire after 7 days.

-  Improve log messages for ``tf.keras`` trial callbacks.

-  Add ``nvidia-container-toolkit`` support.

-  Fix an error in the experimental ``bert_glue_pytorch`` example.

-  The ``tf.keras`` examples for the Native and Trial APIs now refer to the same model.

-  Add a topic guide explaining Determined's approach to :ref:`elastic-infrastructure`.

-  Add a topic guide explaining the Native API (since deprecated).

-  UI: The Determined favicon acquires a small dot when any slots are in use.

-  UI: Fix an issue with command sorting in the WebUI.

-  UI: Fix an issue with badges appearing as the wrong color.

Version 0.12.3
==============

**Release Date:** April 27, 2020

-  Add a tutorial for the new (experimental) Native API.

-  Add support for locally testing experiments via ``det e create --local``.

-  Add :class:`determined.experimental.Determined` class for accessing
   :class:`~determined.experimental.ExperimentReference`, :class:`~determined.experimental.Trial`,
   and :class:`~determined.experimental.Checkpoint` objects.

-  TensorBoard logs now appear under the ``storage_path`` for ``shared_fs`` checkpoint
   configurations.

-  Allow commands, notebooks, shells, and TensorBoards to be killed before they are scheduled.

-  Print container exit reason in trial logs.

-  Choose a better default for the ``--tail`` option of command logs.

-  Add REST API endpoints for trials.

-  Support the execution of a startup script inside the agent Docker container

-  Master and agent Docker containers will have the 'unless-stopped' restart policy by default when
   using ``det-deploy local``.

-  Prevent the ``det trial logs -f`` command from waiting for too long after the trial being watched
   reaches a terminal state.

-  Fix bug where logs disappear when an image is pulled.

-  Fix bug that affected the use of :class:`~determined.pytorch.LRScheduler` in
   :class:`~determined.pytorch.PyTorchTrial` for multi-GPU training.

-  Fix bug after master restart where some errored experiments would show progress indicators.

-  Fix ordering of steps from ``det trial describe --json``.

-  Docs: Added topic guide for effective distributed training.

-  Docs: Reorganize install documentation.

-  UI: Move the authenticated user to the top of the users list filter on the dashboard, right after
   "All".

Version 0.12.2
==============

**Release Date:** April 21, 2020

**Breaking Changes**

-  Rename PEDL to Determined. The canonical way to import it is via ``import determined as det``.

-  Reorganize source code. The frameworks module was removed, and each framework's submodules were
   collapsed into the main framework module. For example:

   -  ``det.frameworks.pytorch.pytorch_trial.PyTorchTrial`` is now ``det.pytorch.PyTorchTrial``
   -  ``det.frameworks.pytorch.data.DataLoader`` is now ``det.pytorch.DataLoader``
   -  ``det.frameworks.pytorch.checkpoint.load`` is now ``det.pytorch.load``
   -  ``det.frameworks.pytorch.util.reset_parameters`` is now ``det.pytorch.reset_parameters``
   -  ``det.frameworks.keras.tf_keras_trial.TFKerasTrial`` is now ``det.keras.TFKerasTrial``
   -  ``det.frameworks.tensorflow.estimator_trial.EstimatorTrial`` is now
      ``det.estimator.EstimatorTrial``
   -  ``det.frameworks.tensorpack.tensorpack_trial`` is now ``det.tensorpack.TensorpackTrial``
   -  ``det.frameworks.util`` and ``det.frameworks.pytorch.util`` have been removed entirely

-  Unify all plugin functions under the Trial class. ``make_data_loaders`` has been moved to two
   functions that should be implemented as part of the Trial class. For example,
   :class:`~determined.pytorch.PyTorchTrial` data loaders should now be implemented in
   ``build_training_data_loader()`` and ``build_validation_data_loader()`` in the trial definition.
   Please see updated examples and documentation for changes in each framework.

-  Trial classes are now required to define a constructor function. The signature of the constructor
   function is:

   .. code:: python

      def __init__(self, context) -> None:
          ...

   where ``context`` is an instance of the new ``det.TrialContext`` class. This new object is the
   primary mechanism for querying information about the system. Some of its methods include:

   -  ``get_hparam(name)``: get a hyperparameter by name
   -  ``get_trial_id()``: get the trial ID being trained
   -  ``get_experiment_config()``: get the experiment config for this experiment
   -  ``get_per_slot_batch_size()``: get the batch size appropriate for training (which will be
      adjusted from the ``global_batch_size`` hyperparameter in distributed training experiments)
   -  ``get_global_batch_size()``: get the effective batch size (which differs from per-slot batch
      size in distributed training experiments)
   -  ``distributed.get_rank()``: get the unique process rank (one process per slot)
   -  ``distributed.get_local_rank()``: get a unique process rank within the agent
   -  ``distributed.get_size()``: get the number of slots
   -  ``distributed.get_num_agents``: get the number of agents (machines) being used

-  The ``global_batch_size`` hyperparameter is required (that is, a hyperparameter with this name
   must be specified in the configuration of every experiment). Previously, the hyperparameter
   ``batch_size`` was required and was manipulated automatically for distributed training. Now
   ``global_batch_size`` will not be manipulated; users should train based on
   ``context.get_per_slot_batch_size()``.

-  Remove ``download_data()``. If users wish to download data at runtime, they should make sure that
   each process (one process per slot) downloads to a unique location. This can be accomplished by
   appending ``context.get_rank()`` to the download path.

-  Remove ``det.trial_controller.util.get_rank()`` and
   ``det.trial_controller.util.get_container_gpus()``. Use ``context.distributed.get_rank()`` and
   ``context.distributed.get_num_agents()`` instead.

**General Improvements**

-  ``tf.data.Dataset`` is now supported as input for all versions of TensorFlow (1.14, 1.15, 2.0,
   2.1) for TFKerasTrial and EstimatorTrial. Please note that Determined currently does not support
   checkpointing ``tf.data.Dataset`` inputs. Therefore, when resuming training, it resumes from the
   start of the dataset. Model weights are loaded correctly as always.

-  ``TFKerasTrial`` now supports five different types of inputs:

   #. A tuple ``(x_train, y_train)`` of NumPy arrays. ``x_train`` must be a NumPy array (or
      array-like), a list of arrays (in case the model has multiple inputs), or a dict mapping input
      names to the corresponding array, if the model has named inputs. ``y_train`` should be a NumPy
      array.

   #. A tuple ``(x_train, y_train, sample_weights)`` of NumPy arrays.

   #. A `tf.data.Dataset <https://www.tensorflow.org/api_docs/python/tf/data/Dataset>`__ returning a
      tuple of either ``(inputs, targets)`` or ``(inputs, targets, sample_weights)``.

   #. A `keras.utils.Sequence
      <https://www.tensorflow.org/api_docs/python/tf/keras/utils/PyDataset>`__ returning a tuple of
      either ``(inputs, targets)`` or ``(inputs, targets, sample weights)``.

   #. A ``det.keras.SequenceAdapter`` returning a tuple of either ``(inputs, targets)`` or
      ``(inputs, targets, sample weights)``.

-  PyTorch trial checkpoints no longer save in MLflow's MLmodel format.

-  The ``det trial download`` command now accepts ``-o`` to save a checkpoint to a specific path.
   PyTorch checkpoints can then be loaded from a specified local file system path.

-  Allow the agent to read configuration values from a YAML file.

-  Include experiment ID in the downloaded trial logs.

-  Display checkpoint storage location in the checkpoint info modal for trials and experiments.

-  Preserve recent tasks' filter preferences in the WebUI.

-  Add task name to ``det slot list`` command output.

-  Model definitions are now downloaded as compressed tarfiles (.tar.gz) instead of zipfiles (.zip).

-  ``startup-hook.sh`` is now executed in the same directory as the model definition.

-  Rename ``projects`` to ``examples`` in the Determined repository.

-  Improve documentation:

   -  Add documentation page on the lifecycle of an experiment.
   -  Add how-to and topic guides for multi-GPU (both for single-machine parallel and multi-machine)
      training.
   -  Add a topic guide on best practices for writing model definitions.

-  Fix bug that occasionally caused multi-machine training to hang on initialization.

-  Fix bug that prevented ``TensorpackTrial`` from successfully loading checkpoints.

-  Fix a bug in ``TFKerasTrial`` where runtime errors could cause the trial to hang or would
   silently drop the stack trace produced by Keras.

-  Fix trial lifecycle bugs for containers that exit during the pulling phase.

-  Fix bug that led to some distributed trials timing out.

-  Fix bug that caused ``tf.keras`` trials to fail in the multi-GPU setting when using an optimizer
   specified by its name.

-  Fix bug in the CLI for downloading model definitions.

-  Fix performance issues for experiments with very large numbers of trials.

-  Optimize performance for scheduling large hyperparameter searches.

-  Add configuration for telemetry in ``master.yaml``.

-  Add a utility function for initializing a trial class for development (det.create_trial_instance)

-  Add `security.txt <https://securitytxt.org/>`__.

-  Add ``det.estimator.load()`` to load TensorFlow Estimator ``saved_model`` checkpoints into
   memory.

-  Ensure AWS EC2 keypair exists in account before creating the CloudFormation stack.

-  Add support for gradient aggregation in Keras trials for TensorFlow 2.1.

-  Add Trial and Checkpoint experimental APIs for exporting and loading checkpoints.

-  Improve performance when starting many tasks simultaneously.

**Web Improvements**

-  Improve discoverability of dashboard actions.
-  Add dropdown action menu for killing and archiving recent tasks on the dashboard.
-  Add telemetry for web interactions.
-  Fix an issue around cluster utilization status showing as "No Agent" for a brief moment during
   initial load.
-  Add Ace editor to attributions list.
-  Set UI preferences based on the logged-in user.
-  Fix an issue where the indicated user filter was not applied to the displayed tasks.
-  Improve error messaging for failed actions.
