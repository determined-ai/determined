:orphan:

.. _release-notes:

###############
 Release Notes
###############

**************
 Version 0.19
**************

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

-  API: `GetTrialWorkloads` can now optionally include per-batch metrics when
   ``includeBatchMetrics`` query parameter is set.

**New Features**

-  Cluster: The enterprise edition of Determined ([HPE Machine Learning Development
   Environment](https://www.hpe.com/us/en/solutions/artificial-intelligence/machine-learning-development-environment.html)),
   can now be deployed on a Slurm cluster. When using Slurm, Determined delegates all job scheduling
   and prioritization to the Slurm workload manager. This integration enables existing Slurm
   workloads and Determined workloads to coexist and access all of the advanced capabilities of the
   Slurm workload manager. The Determined Slurm integration can use either Singularity or Podman for
   the container runtime.

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
   See `video <https://youtu.be/zJP7p0CWubw>`_ for a walkthrough.

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
   JSON-formatted notebook, shell or tensorboard task list.
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

   .. image:: https://user-images.githubusercontent.com/220971/169450323-d169f4ee-2698-4ae8-9b1a-c04460751310.png

**Improvements**

-  Security: Improved security by requiring admin privileges for the following actions.

   -  Reading master config.
   -  Enabling or disabling an agent.
   -  Enabling or disabling a slot.

-  Logging: Ensure logs for very short tasks are not truncated in Kubernetes.

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
   <https://yogadl.readthedocs.io>`__ directly before removing the feature.

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
      Previously, the associated docker image lacked dependencies for panoptic segmentation. Users
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

   .. image:: https://user-images.githubusercontent.com/15078396/152874240-6365b276-3f3e-4fb6-aa2b-0cedc7451b12.png

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

   .. image:: https://user-images.githubusercontent.com/15078396/144926881-98aeb187-aa3f-4e40-b502-d7af624573db.png

   .. image:: https://user-images.githubusercontent.com/15078396/144926889-eec0216a-dacc-4fe5-ac28-858ea6587d04.png

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

   .. image:: https://user-images.githubusercontent.com/220971/133659677-aea0d1bc-95ce-4652-8218-92b97d114358.png

-  WebUI: Display the latest log entry available for a trial at the bottom of the trial's page. This
   works for both single-trial experiments and trials within a multi-trial experiment.

   .. image:: https://user-images.githubusercontent.com/220971/131391658-4be1a1f4-1d46-4766-a737-7eb8efcb65b4.png

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
   <https://github.com/pytorch/examples/tree/master/word_language_model>`__ as a `Determined example
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
   `HuggingFace transformers library for NLP <https://huggingface.co/transformers>`__.

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
   and :ref:`estimators <estimators-custom-reducers>` from experimental status to general
   availability.

-  Resource pools: Support configuring distinct ``task_container_defaults`` for each resource pool
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
   errors on non-numeric Numpy-like data. As PyTorch is still unable to move such data to the GPU,
   non-numeric arrays will simply remain on the CPU. This is especially useful to NLP practitioners
   who wish to make use of Numpy's string manipulations anywhere in their data pipelines.

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
   <https://github.com/determined-ai/determined/tree/master/examples/computer_vision/deformabledetr_coco_pytorch>`__
   in the Determined repository.

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
   <https://developers.google.com/protocol-buffers/docs/reference/google.protobuf#google.protobuf.Struct>`__.

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
-  Support Elasticsearch as an alternative backend for logging. Read more about
   :ref:`elasticsearch-logging-backend` to see if it's appropriate for your Determined deployment.

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

   See ``determined/examples/features/custom_reducers_mnist_pytorch`` for a complete example of how
   to use custom reducers. The example emits a per-class F1 score using the new custom reducer API.

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
   <https://github.com/determined-ai/determined/tree/master/examples/computer_vision/mmdetection_pytorch>`__
   for more information on how to get started.

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
-  Add support for Nvidia T4 GPUs.
-  ``det-deploy``: Add support for ``g4`` instance types on AWS.
-  Upgrade Nvidia drivers on the default AWS and GCP images from ``410.104`` to ``450.51.05``.

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
   <https://github.com/determined-ai/determined/tree/master/examples/experimental/trial/unets_tf_keras>`__
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
   associated with them. Metadata can be added, queried, and removed via a :class:`Python API
   <determined.experimental.Checkpoint>`.

-  Add support for Keras callbacks that stop training early, including the `official EarlyStopping
   callback <https://www.tensorflow.org/api_docs/python/tf/keras/callbacks/EarlyStopping>`__. When a
   stop is requested, Determined will finish the training (or validation) step we are in,
   checkpoint, and terminate the trial.

-  Add support for Estimator callbacks that stop training early, including the official
   `stop_if_no_decrease_hook
   <https://www.tensorflow.org/api_docs/python/tf/estimator/experimental/stop_if_no_decrease_hook>`__.
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
   :class:`~determined.experimental.ExperimentReference`,
   :class:`~determined.experimental.TrialReference`, and
   :class:`~determined.experimental.Checkpoint` objects.

-  TensorBoard logs now appear under the ``storage_path`` for ``shared_fs`` checkpoint
   configurations.

-  Allow commands, notebooks, shells, and TensorBoards to be killed before they are scheduled.

-  Print container exit reason in trial logs.

-  Choose a better default for the ``--tail`` option of command logs.

-  Add REST API endpoints for trials.

-  Support the execution of a startup script inside the agent docker container

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

   #. A `keras.utils.Sequence <https://tensorflow.org/api_docs/python/tf/keras/utils/Sequence>`__
      returning a tuple of either ``(inputs, targets)`` or ``(inputs, targets, sample weights)``.

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

-  Add TrialReference and Checkpoint experimental APIs for exporting and loading checkpoints.

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
