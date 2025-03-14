.. _experiment-config-reference:

.. _experiment-configuration:

####################################
 Experiment Configuration Reference
####################################

.. meta::
   :description: Browse this complete description of the experiment configuration reference or YAML file, including metadata, entrypoint, basic behaviors, validation policy, checkpoint policy, checkpoint storage, and so on.

The behavior of an experiment is configured via a YAML file. A configuration file is typically
passed as a command-line argument when an experiment is created with the Determined CLI. For
example:

.. code::

   det experiment create config-file.yaml model-directory

**********
 Metadata
**********

``name``
========

Optional. A short human-readable name for the experiment.

``description``
===============

Optional. A human-readable description of the experiment. This does not need to be unique but should
be limited to less than 255 characters for the best experience.

``labels``
==========

Optional. A list of label names (strings). Assigning labels to experiments allows you to identify
experiments that share the same property or should be grouped together. You can add and remove
labels using either the CLI (``det experiment label``) or the WebUI.

.. _experiment-config-data:

``data``
========

Optional. This field can be used to specify information about how the experiment accesses and loads
training data. The content and format of this field is user-defined: it should be used to specify
whatever configuration is needed for loading data for use by the experiment's model definition. For
example, if your experiment loads data from Amazon S3, the ``data`` field might contain the S3
bucket name, object prefix, and AWS authentication credentials.

As a special case, values found under a subfield named ``secrets`` will be obfuscated when
experiment details are reviewed. For example, given the following configuration:

.. code:: yaml

   name: mnist_tf_const
   data:
      base_url: https://s3-us-west-2.amazonaws.com/determined-ai-datasets/mnist/
      secrets:
         auth_token: f020572a-a847-4cc6-9c2b-625c43515759

The value of ``data["secrets"]["auth_token"]`` will be usable during the experiment run, but not
when users view the experiment configuration. Note these values may still be visible in the
configuration file itself; to hide this file from model context, add it to a ``.detignore`` file
(see :ref:`Creating an Experiment <creating-an-experiment>`).

See also: :ref:`det API Reference <det-reference>` > ``user_data`` property.

``workspace``
=============

Optional. The name of the pre-existing workspace where you want to create the experiment. The
``workspace`` and ``project`` fields must either both be present or both be absent. If they are
absent, the experiment is placed in the ``Uncategorized`` project in the ``Uncategorized``
workspace. You can manage workspaces using the CLI ``det workspace help`` command or the WebUI.

``project``
===========

Optional. The name of the pre-existing project inside ``workspace`` where you want to create the
experiment. The ``workspace`` and ``project`` fields must either both be present or both be absent.
If they are absent, the experiment is placed in the ``Uncategorized`` project in the
``Uncategorized`` workspace. You can manage projects using the CLI ``det project help`` command or
the WebUI.

************
 Entrypoint
************

.. _experiment-config-entrypoint:

``entrypoint``
==============

Required. A model definition trial class specification or Python launcher script, which is the model
processing entrypoint. This field can have the following formats.

Formats that specify a trial class have the form ``<module>:<object_reference>``.

The ``<module>`` field specifies the module containing the trial class in the model definition,
relative to root.

The ``<object_reference>`` specifies the trial class name in the module, which can be a nested
object delimited by a period (``.``).

Examples:

-  ``MnistTrial`` expects an *MnistTrial* class exposed in a ``__init__.py`` file at the top level
   of the context directory.
-  ``model_def:CIFAR10Trial`` expects a *CIFAR10Trial* class defined in the ``model_def.py`` file at
   the top level of the context directory.
-  ``determined_lib.trial:trial_classes.NestedTrial`` expects a ``NestedTrial`` class, which is an
   attribute of ``trial_classes`` defined in the ``determined_lib/trial.py`` file.

These formats follow Python `Entry points
<https://packaging.python.org/en/latest/specifications/entry-points/>`_ specification except that
the context directory name is prefixed by ``<module>`` or used as the module if the ``<module>``
field is empty.

Arbitrary Script
----------------

Required. An arbitrary entrypoint script with args.

Example:

.. code:: yaml

   entrypoint: ./hello.sh args...

Preconfigured Launch Module with Script
---------------------------------------

Required. The name of a preconfigured launch module and script with args.

Example:

.. code:: yaml

   entrypoint: python3 -m (LAUNCH_MODULE) train.py args...

``LAUNCH_MODULE`` options:

-  Horovod (determined.launch.horovod)
-  PyTorch (determined.launch.torch_distributed)
-  Deepspeed (determined.launch.deepspeed)
-  TensorFlow (determined.launch.tensorflow)

Preconfigured Launch Module with Legacy Trial Definition
--------------------------------------------------------

Required. The name of a preconfigured launch module and legacy trial class specification.

Example:

.. code:: yaml

   entrypoint: python3 -m (LAUNCH_MODULE) --trial model_def:Trial

``LAUNCH_MODULE`` options: [need literals for these]

-  Horovod (determined.launch.horovod)
-  PyTorch (determined.launch.torch_distributed)
-  Deepspeed (determined.launch.deepspeed)

Legacy Trial Definition
-----------------------

Required. A legacy trial class specification.

Example:

.. code:: yaml

   entrypoint: model_def:Trial

*****************
 Basic Behaviors
*****************

.. _scheduling-unit:

``scheduling_unit`` (deprecated)
================================

Optional. Instructs how frequent to perform system operations, such as periodic checkpointing and
preemption, in the unit of batches. This field has been deprecated and the behavior should be
configured in training code directly. Please see :ref:`apis-howto-overview` for details specific to
your training framework.

.. _config-records-per-epoch:

``records_per_epoch`` (deprecated)
==================================

Optional. The number of records in the training data set. This field has been deprecated.

.. _max-restarts:

``max_restarts``
================

Optional. The ``max_restarts`` parameter parameter sets a limit on the number of times the
Determined master can try restarting a trial, preventing an infinite loop if the same error
repeatedly occurs. After reach the ``max_restarts`` limit for an experiment, any subsequent failed
trials will not be restarted and will be marked as errored. An experiment is considered successful
if at least one of its trials completes without errors. The default value for ``max_restarts`` is
``5``.

.. _config-log-policies:

``log_policies``
================

Optional. Defines actions and labels in response to trial logs matching specified regex patterns (Go
language syntax). For more information about the syntax, you can visit this `RE2 reference page
<https://github.com/google/re2/wiki/Syntax>`__. Each log policy can have the following fields:

-  ``name``: Required. The name of the log policy, displayed as a label in the WebUI when a log
   policy match occurs.

-  ``pattern``: Optional. Defines a regex pattern to match log entries. If not specified, this
   policy is disabled.

-  ``action``: Optional. The action to take when the pattern is matched. Actions include:

   -  ``exclude_node``: Excludes a failed trial's restart attempts (due to its ``max_restarts``
      policy) from being scheduled on nodes with matched error logs. This is useful for bypassing
      nodes with hardware issues, such as uncorrectable GPU ECC errors.

      .. note::

         This option is not supported on PBS systems.

      For the agent resource manager, if a trial becomes unschedulable due to enough node
      exclusions, and ``launch_error`` in the master config is set to true (default), the trial will
      fail.

   -  ``cancel_retries``: Prevents a trial from restarting if a log matches the pattern, even if the
      trial has remaining max_restarts. This avoids using resources retrying a trial that encounters
      failures unlikely to be resolved by retrying, such as CUDA memory issues.

Example configuration:

.. code:: yaml

   log_policies:
     - name: ECC Error
       pattern: ".*uncorrectable ECC error encountered.*"
       action: exclude_node
     - name: CUDA OOM
       pattern: ".*CUDA out of memory.*"
       action: cancel_retries

When a log policy matches, its name appears as a label in the WebUI, making it easy to identify
specific issues during a run. These labels are shown in both the run table and run detail views.

These settings may also be specified at the cluster or resource pool level through task container
defaults.

Default policies:

.. code:: yaml

   log_policies:
     - name: CUDA OOM
       pattern: ".*CUDA out of memory.*"
     - name: ECC Error
       pattern: ".*uncorrectable ECC error encountered.*"

To disable showing labels from the default policies:

.. code:: yaml

   log_policies:
     - name: CUDA OOM
     - name: ECC Error

.. _log-retention-days:

``retention_policy``
====================

Optional. Defines retention policies for logs related to all trials of a given experiment.
Parameters include:

-  ``log_retention_days``: Optional. Overrides the number of days to retain logs for a trial set in
   the cluster's task container defaults. Acceptable values range from ``-1`` to ``32767``. If set
   to ``-1``, logs will be retained indefinitely. If set to ``0``, logs will be deleted during the
   next cleanup. To modify the retention settings post-completion for a single trial or the entire
   experiment, you can use the CLI command ``det t set log-retention <trial-id>`` or ``det e set
   log-retention <exp-id>``. Both commands accept either the argument: ``--days``, which sets the
   number of days to retain logs from the end time of the task, or ``--forever`` which retains logs
   indefinitely.

   Note: If the cluster's log retention policy/days is upgraded after the experiment is created, the
   new cluster value will override the old experiment value.

Example configuration:

.. code:: yaml

   retention_policy:
      log_retention_days: 90

This setting can be defined as a default setting for the entire cluster.

**********************************************
 ``debug`` option in agent configuration file
**********************************************

The :ref:`debug <agent-config-ref-debug>` option in the agent configuration file enables more
verbose logging for diagnostic purposes when set to ``true``.

While debugging, the logger will display lines highlighted in blue for easy identification.

*******************
 Validation Policy
*******************

.. _experiment-config-min-validation-period:

``min_validation_period`` (deprecated)
======================================

Optional. Specifies the minimum frequency at which validation should be run for each trial. This
field has been deprecated and should be specified directly in training code. Please see
:ref:`apis-howto-overview` for details specific to your training framework.

.. _experiment-config-perform-initial-validation:

``perform_initial_validation``
==============================

Optional. Instructs Determined to perform an initial validation before any training begins, for each
trial. This can be useful to determine a baseline when fine-tuning a model on a new dataset.

.. _experiment-config-checkpoint-policy:

*******************
 Checkpoint Policy
*******************

Determined checkpoints in the following situations:

-  Periodically during training, to keep a record of the training progress.
-  During training, to enable recovery of the trial's execution in case of resumption or errors.
-  Upon completion of the trial.
-  Prior to the searcher making a decision based on the validation of trials, ensuring consistency
   in case of a failure.

.. _experiment-config-min-checkpoint-period:

``min_checkpoint_period`` (deprecated)
======================================

Optional. Specifies the minimum frequency for running checkpointing for each trial. This field has
been deprecated and should be specified directly in training code. Please see
:ref:`apis-howto-overview` for details specific to your training framework.

``checkpoint_policy``
=====================

Optional. Controls how Determined performs checkpoints after validation operations, if at all.
Should be set to one of the following values:

-  ``best`` (default): A checkpoint will be taken after every validation operation that performs
   better than all previous validations for this experiment. Validation metrics are compared
   according to the ``metric`` and ``smaller_is_better`` options in the :ref:`searcher configuration
   <experiment-configuration_searcher>`.

-  ``all``: A checkpoint will be taken after every validation, no matter the validation performance.

-  ``none``: A checkpoint will never be taken *due* to a validation. However, even with this policy
   selected, checkpoints are still expected to be taken after the trial is finished training, due to
   cluster scheduling decisions, or when specified in training code.

.. _checkpoint-storage:

********************
 Checkpoint Storage
********************

The ``checkpoint_storage`` section defines how model checkpoints will be stored. A checkpoint
contains the architecture and weights of the model being trained. Each checkpoint has a UUID, which
is used as the name of the checkpoint directory on the external storage system.

If this field is not specified, the experiment will default to the checkpoint storage configured in
the :ref:`master configuration <master-config-reference>`.

.. _checkpoint-garbage-collection:

Checkpoint Garbage Collection
=============================

When an experiment finishes, the system will optionally delete some checkpoints to reclaim space.
The ``save_experiment_best``, ``save_trial_best`` and ``save_trial_latest`` parameters specify which
checkpoints to save. If multiple ``save_*`` parameters are specified, the union of the specified
checkpoints are saved.

``save_experiment_best``
------------------------

The number of the best checkpoints with validations over all trials to save (where best is measured
by the validation metric specified in the searcher configuration).

``save_trial_best``
-------------------

The number of the best checkpoints with validations of each trial to save.

``save_trial_latest``
---------------------

The number of the latest checkpoints of each trial to save.

Checkpoint Saving Policy
========================

The checkpoint garbage collection fields default to the following values:

.. code:: yaml

   save_experiment_best: 0
   save_trial_best: 1
   save_trial_latest: 1

This policy will save the most recent *and* the best checkpoint per trial. In other words, if the
most recent checkpoint is also the *best* checkpoint for a given trial, only one checkpoint will be
saved for that trial. Otherwise, two checkpoints will be saved.

Examples
--------

Suppose an experiment has the following trials, checkpoints and validation metrics (where
``smaller_is_better`` is true):

+--------+-------------+-----------------+
| Trial  | Checkpoint  | Validation      |
| ID     | ID          | Metric          |
+========+=============+=================+
| 1      | 1           | null            |
+--------+-------------+-----------------+
| 1      | 2           | null            |
+--------+-------------+-----------------+
| 1      | 3           | 0.6             |
+--------+-------------+-----------------+
| 1      | 4           | 0.5             |
+--------+-------------+-----------------+
| 1      | 5           | 0.4             |
+--------+-------------+-----------------+
| 2      | 6           | null            |
+--------+-------------+-----------------+
| 2      | 7           | 0.2             |
+--------+-------------+-----------------+
| 2      | 8           | 0.3             |
+--------+-------------+-----------------+
| 2      | 9           | null            |
+--------+-------------+-----------------+
| 2      | 10          | null            |
+--------+-------------+-----------------+

The effect of various policies is enumerated in the following table:

+--------------------------+---------------------+-----------------------+----------------------+
| ``save_experiment_best`` | ``save_trial_best`` | ``save_trial_latest`` | Saved Checkpoint IDs |
+==========================+=====================+=======================+======================+
| 0                        | 0                   | 0                     | none                 |
+--------------------------+---------------------+-----------------------+----------------------+
| 2                        | 0                   | 0                     | 8,7                  |
+--------------------------+---------------------+-----------------------+----------------------+
| >= 5                     | 0                   | 0                     | 8,7,5,4,3            |
+--------------------------+---------------------+-----------------------+----------------------+
| 0                        | 1                   | 0                     | 7,5                  |
+--------------------------+---------------------+-----------------------+----------------------+
| 0                        | >= 3                | 0                     | 8,7,5,4,3            |
+--------------------------+---------------------+-----------------------+----------------------+
| 0                        | 0                   | 1                     | 10,5                 |
+--------------------------+---------------------+-----------------------+----------------------+
| 0                        | 0                   | 3                     | 10,9,8,5,4,3         |
+--------------------------+---------------------+-----------------------+----------------------+
| 2                        | 1                   | 0                     | 8,7,5                |
+--------------------------+---------------------+-----------------------+----------------------+
| 2                        | 0                   | 1                     | 10,8,7,5             |
+--------------------------+---------------------+-----------------------+----------------------+
| 0                        | 1                   | 1                     | 10,7,5               |
+--------------------------+---------------------+-----------------------+----------------------+
| 2                        | 1                   | 1                     | 10,8,7,5             |
+--------------------------+---------------------+-----------------------+----------------------+

If aggressive reclamation is desired, set ``save_experiment_best`` to a 1 or 2 and leave the other
parameters zero. For more conservative reclamation, set ``save_trial_best`` to 1 or 2; optionally
set ``save_trial_latest`` as well.

Checkpoints of an existing experiment can be garbage collected by changing the GC policy using the
``det experiment set gc-policy`` subcommand of the Determined CLI.

**************
 Storage Type
**************

Determined currently supports several kinds of checkpoint storage, ``gcs``, ``s3``, ``azure``, and
``shared_fs``, identified by the ``type`` subfield. Additional fields may also be required,
depending on the type of checkpoint storage in use. For example, to store checkpoints on Google
Cloud Storage:

.. code:: yaml

   checkpoint_storage:
     type: gcs
     bucket: <your-bucket-name>

Google Cloud Storage
====================

If ``type: gcs`` is specified, checkpoints will be stored on Google Cloud Storage (GCS).
Authentication is done using GCP's "`Application Default Credentials
<https://googleapis.dev/python/google-api-core/latest/auth.html>`__" approach. When using Determined
inside Google Compute Engine (GCE), the simplest approach is to ensure that the VMs used by
Determined are running in a service account that has the "Storage Object Admin" role on the GCS
bucket being used for checkpoints. As an alternative (or when running outside of GCE), you can add
the appropriate `service account credentials
<https://cloud.google.com/docs/authentication/set-up-adc-attached-service-account>`__ to your
container (e.g., via a bind-mount), and then set the ``GOOGLE_APPLICATION_CREDENTIALS`` environment
variable to the container path where the credentials are located. See :ref:`environment-variables`
for more details on how to set environment variables in containers.

``bucket``
----------

Required. The GCS bucket name to use.

``prefix``
----------

Optional. The optional path prefix to use. Must not contain ``..``. Note: Prefix is normalized,
e.g., ``/pre/.//fix`` -> ``/pre/fix``

Amazon S3
=========

If ``type: s3`` is specified, checkpoints will be stored in Amazon S3 or an S3-compatible object
store such as `MinIO <https://min.io/>`__.

``bucket``
----------

Required. The S3 bucket name to use.

``access_key``
--------------

Required. The AWS access key to use.

``secret_key``
--------------

Required. The AWS secret key to use.

``prefix``
----------

Optional. The optional path prefix to use. Must not contain ``..``. Note: Prefix is normalized,
e.g., ``/pre/.//fix`` -> ``/pre/fix``

``endpoint_url``
----------------

Optional. The endpoint to use for S3 clones, e.g., ``http://127.0.0.1:8080/``. If not specified,
Amazon S3 will be used.

Azure Blob Storage
==================

If ``type: azure`` is specified, checkpoints will be stored in Microsoft's Azure Blob Storage.

Please only specify one of ``connection_string`` or the ``account_url``, ``credential`` tuple.

``container``
-------------

Required. The Azure Blob Storage container name to use.

``connection_string``
---------------------

Required. The connection string for the Azure Blob Storage service account to use.

``account_url``
---------------

Required. The account URL for the Azure Blob Storage service account to use.

``credential``
--------------

Optional. The credential to use with the ``account_url``.

Shared File System
==================

If ``type: shared_fs`` is specified, checkpoints will be written to a directory on the agent's file
system. The assumption is that the system administrator has arranged for the same directory to be
mounted at every agent machine, and for the content of this directory to be the same on all agent
hosts (e.g., by using a distributed or network file system such as `GlusterFS
<https://www.gluster.org/>`__ or `NFS <https://en.wikipedia.org/wiki/Network_File_System>`__).

.. warning::

   When downloading checkpoints from a shared file system (e.g., using ``det checkpoint download``),
   we assume the same shared file system is mounted locally at the same ``host_path``.

``host_path``
-------------

Required. The file system path on each agent to use. This directory will be mounted to
``/determined_shared_fs`` inside the trial container.

**Optional Fields**

``storage_path``
----------------

Optional. The path where checkpoints will be written to and read from. Must be a subdirectory of the
``host_path`` or an absolute path containing the ``host_path``. If not specified, checkpoints are
written to and read from the ``host_path``.

``propagation``
---------------

Optional. `Propagation behavior
<https://docs.docker.com/engine/storage/bind-mounts/#configure-bind-propagation>`__ for replicas of
the bind-mount. Defaults to ``rprivate``.

Local Directory
===============

If ``type: directory`` is specified, checkpoints will be written to a local directory. For tasks
running on Determined platform, it's a path within the container. For detached mode, it's simply a
local path.

The assumption is that a persistent storage will be mounted at the path parametrized by
``container_path`` option using ``bind_mounts``, ``pod_spec``, or other mechanisms. Otherwise, this
path will usually end up being ephemeral storage within the container, and the data will be lost
when the container exits.

.. warning::

   TensorBoards currently do not inherit ``bind_mounts`` or ``pod_specs`` from their parent
   experiments. Therefore, if an experiment is using ``type: directory`` storage, and mounts the
   storage separately, a launched TensorBoard will need the same mount configuration provided
   explicitly using ``det tensorboard start <experiment_id> --config-file <CONFIG FILE>`` or
   similar.

.. warning::

   When downloading checkpoints (e.g., using ``det checkpoint download``), Determined assumes the
   same directory is present locally at the same ``container_path``.

``container_path``
------------------

Required. The file system path to use.

.. _experiment-configuration_hyperparameters:

*****************
 Hyperparameters
*****************

The ``hyperparameters`` section defines the hyperparameter space for the experiment. The appropriate
hyperparameters for a specific model depend on the nature of the model being trained. In Determined,
it is common to specify hyperparameters that influence various aspects of the model's behavior, such
as data augmentation, neural network architecture, and the choice of optimizer, as well as its
configuration.

To access the value of a hyperparameter in a particular trial, use the trial context with
:func:`context.get_hparam() <determined.TrialContext.get_hparam>`. For example, you can access the
current value of a hyperparameter named ``learning_rate`` by calling
``context.get_hparam("learning_rate")``.

.. _config-global-batch-size:

.. note::

   Every experiment must specify a hyperparameter called ``global_batch_size``. This hyperparameter
   is required for distributed training to calculate the appropriate per-worker batch size. The
   batch size per slot is computed at runtime, based on the number of slots used to train a single
   trial of the experiment (see :ref:`resources.slots_per_trial
   <exp-config-resources-slots-per-trial>`). To access the updated values, use the trial context
   with :func:`context.get_per_slot_batch_size() <determined.TrialContext.get_per_slot_batch_size>`
   and :func:`context.get_global_batch_size() <determined.TrialContext.get_global_batch_size>`.

.. include:: ../_shared/note-dtrain-learn-more.txt

The hyperparameter space is defined by a dictionary. Each key in the dictionary is the name of a
hyperparameter; the associated value defines the range of the hyperparameter. If the value is a
scalar, the hyperparameter is a constant; otherwise, the value should be a nested map. Here is an
example:

.. code:: yaml

   hyperparameters:
     global_batch_size: 64
     optimizer_config:
       optimizer:
         type: categorical
         vals:
           - SGD
           - Adam
           - RMSprop
       learning_rate:
         type: log
         minval: -5.0
         maxval: 1.0
         base: 10.0
     num_layers:
       type: int
       minval: 1
       maxval: 3
     layer1_dropout:
       type: double
       minval: 0.2
       maxval: 0.5

This configuration defines the following hyperparameters:

-  ``global_batch_size``: a constant value

-  ``optimizer_config``: a top level nested hyperparameter with two child hyperparameters:

   -  ``optimizer``: a categorical hyperparameter
   -  ``learning_rate``: a log scale hyperparameter

-  ``num_layers``: an integer hyperparameter

-  ``layer1_dropout``: a double hyperparameter

The field ``optimizer_config`` demonstrates how nesting can be used to organize hyperparameters.
Arbitrary levels of nesting are supported with all types of hyperparameters. Aside from
hyperparameters with constant values, the four types of hyperparameters -- ``categorical``,
``double``, ``int``, and ``log`` -- can take on a range of possible values. The following sections
cover how to configure the hyperparameter range for each type of hyperparameter.

Categorical
===========

A ``categorical`` hyperparameter ranges over a set of specified values. The possible values are
defined by the ``vals`` key. ``vals`` is a list; each element of the list can be of any valid YAML
type, such as a boolean, a string, a number, or a collection.

Double
======

A ``double`` hyperparameter is a floating point variable. The minimum and maximum values of the
variable are defined by the ``minval`` and ``maxval`` keys, respectively (inclusive of endpoints).

When doing a grid search, the ``count`` key must also be specified; this defines the number of
points in the grid for this hyperparameter. Grid points are evenly spaced between ``minval`` and
``maxval``. See :ref:`topic-guides_hp-tuning-det_grid` for details.

Integer
=======

An ``int`` hyperparameter is an integer variable. The minimum and maximum values of the variable are
defined by the ``minval`` and ``maxval`` keys, respectively (inclusive of endpoints).

When doing a grid search, the ``count`` key must also be specified; this defines the number of
points in the grid for this hyperparameter. Grid points are evenly spaced between ``minval`` and
``maxval``. See :ref:`topic-guides_hp-tuning-det_grid` for details.

Log
===

A ``log`` hyperparameter is a floating point variable that is searched on a logarithmic scale. The
base of the logarithm is specified by the ``base`` field; the minimum and maximum exponent values of
the hyperparameter are given by the ``minval`` and ``maxval`` fields, respectively (inclusive of
endpoints).

When doing a grid search, the ``count`` key must also be specified; this defines the number of
points in the grid for this hyperparameter. Grid points are evenly spaced between ``minval`` and
``maxval``. See :ref:`topic-guides_hp-tuning-det_grid` for details.

.. _experiment-configuration_searcher:

**********
 Searcher
**********

The ``searcher`` section defines how the experiment's hyperparameter space will be explored. To run
an experiment that trains a single trial with fixed hyperparameters, specify the ``single`` searcher
and specify constant values for the model's hyperparameters. Otherwise, Determined supports three
different hyperparameter search algorithms: ``adaptive_asha``, ``random``, and ``grid``.

The name of the hyperparameter search algorithm to use is configured via the ``name`` field; the
remaining fields configure the behavior of the searcher and depend on the searcher being used. For
example, to configure a ``random`` hyperparameter search that trains 5 trials for 1000 batches each:

.. code:: yaml

   searcher:
     name: random
     metric: accuracy
     max_trials: 5

For details on using Determined to perform hyperparameter search, refer to
:ref:`hyperparameter-tuning`. For more information on the search methods supported by Determined,
refer to :ref:`hyperparameter-tuning`.

Single
======

The ``single`` search method does not perform a hyperparameter search at all; rather, it trains a
single trial for a fixed length. When using this search method, all of the hyperparameters specified
in the :ref:`hyperparameters <experiment-configuration_hyperparameters>` section must be constants.

``metric``
----------

Required. The name of the validation metric used to evaluate the performance of a hyperparameter
configuration.

.. _experiment-configuration_single-searcher-max-length:

``max_length`` (deprecated)
---------------------------

Previously, ``max_length`` was required to determine the length of each trial. This field has been
deprecated and all training lengths should be specified directly in training code.

**Optional Fields**

``smaller_is_better``
---------------------

Optional. Whether to minimize or maximize the metric defined above. The default value is ``true``
(minimize).

``source_trial_id``
-------------------

Optional. If specified, the weights of this trial will be initialized to the most recent checkpoint
of the given trial ID. This will fail if the source trial's model architecture is inconsistent with
the model architecture of this experiment.

``source_checkpoint_uuid``
--------------------------

Optional. Like ``source_trial_id``, but specifies an arbitrary checkpoint from which to initialize
weights. At most one of ``source_trial_id`` or ``source_checkpoint_uuid`` should be set.

Random
======

The ``random`` search method implements a simple random search. The user specifies how many
hyperparameter configurations should be trained and how long each configuration should be trained
for; the configurations are sampled randomly from the hyperparameter space.

``metric``
----------

Required. The name of the validation metric used to evaluate the performance of a hyperparameter
configuration.

``max_trials``
--------------

Required. The number of trials, i.e., hyperparameter configurations, to evaluate.

``max_length`` (deprecated)
---------------------------

Previously, ``max_length`` was required to determine the length of each trial. This field has been
deprecated and all training lengths should be specified directly in training code.

**Optional Fields**

``smaller_is_better``
---------------------

Optional. Whether to minimize or maximize the metric defined above. The default value is ``true``
(minimize).

``max_concurrent_trials``
-------------------------

Optional. The maximum number of trials that can be worked on simultaneously. The default value is
``16``. When the value is ``0`` we will work on as many trials as possible.

``source_trial_id``
-------------------

Optional. If specified, the weights of *every* trial in the search will be initialized to the most
recent checkpoint of the given trial ID. This will fail if the source trial's model architecture is
incompatible with the model architecture of any of the trials in this experiment.

``source_checkpoint_uuid``
--------------------------

Optional. Like ``source_trial_id`` but specifies an arbitrary checkpoint from which to initialize
weights. At most one of ``source_trial_id`` or ``source_checkpoint_uuid`` should be set.

Grid
====

The ``grid`` search method performs a grid search. The coordinates of the hyperparameter grid are
specified via the ``hyperparameters`` field. For more details see the
:ref:`topic-guides_hp-tuning-det_grid`.

``metric``
----------

Required. The name of the validation metric used to evaluate the performance of a hyperparameter
configuration.

``max_length`` (deprecated)
---------------------------

Previously, ``max_length`` was required to determine the length of each trial. This field has been
deprecated and all training lengths should be specified directly in training code.

**Optional Fields**

``smaller_is_better``
---------------------

Optional. Whether to minimize or maximize the metric defined above. The default value is ``true``
(minimize).

``max_concurrent_trials``
-------------------------

Optional. The maximum number of trials that can be worked on simultaneously. The default value is
``16``. When the value is ``0`` we will work on as many trials as possible.

``source_trial_id``
-------------------

Optional. If specified, the weights of this trial will be initialized to the most recent checkpoint
of the given trial ID. This will fail if the source trial's model architecture is inconsistent with
the model architecture of this experiment.

``source_checkpoint_uuid``
--------------------------

Optional. Like ``source_trial_id``, but specifies an arbitrary checkpoint from which to initialize
weights. At most one of ``source_trial_id`` or ``source_checkpoint_uuid`` should be set.

.. _experiment-configuration-searcher-asha:

Asynchronous Halving (ASHA)
===========================

The ``async_halving`` search performs a version of the asynchronous successive halving algorithm
(`ASHA <https://arxiv.org/pdf/1810.05934.pdf>`_) that stops trials early if there is enough evidence
to terminate training. Once trials are stopped, they will not be resumed.

``metric``
----------

Required. The name of the validation metric used to evaluate the performance of a hyperparameter
configuration.

``max_length`` (deprecated)
---------------------------

The length of the trial. This field has been deprecated and should be replaced with ``time_metric``
and ``max_time`` below.

``time_metric``
---------------

Required. The name of the validation metric used to evaluate the progress of a given trial.

``max_time``
------------

Required. The maximum value that ``time_metric`` should take when a trial finishes training. Early
stopping is decided based on how far the ``time_metric`` has progressed towards this ``max_time``
value.

``max_trials``
--------------

Required. The number of trials, i.e., hyperparameter configurations, to evaluate.

``num_rungs``
-------------

Required. The number of rounds of successive halving to perform.

``smaller_is_better``
---------------------

Optional. Whether to minimize or maximize the metric defined above. The default value is ``true``
(minimize).

``divisor``
-----------

Optional. The fraction of trials to keep at each rung, and also determines the training length for
each rung. The default setting is ``4``; only advanced users should consider changing this value.

``max_concurrent_trials``
-------------------------

Optional. The maximum number of trials that can be worked on simultaneously. The default value is
``16``, and we set reasonable values depending on ``max_trials`` and the number of rungs in the
brackets. This is akin to controlling the degree of parallelism of the experiment. If this value is
less than the number of brackets produced by the adaptive algorithm, it will be rounded up.

``source_trial_id``
-------------------

Optional. If specified, the weights of *every* trial in the search will be initialized to the most
recent checkpoint of the given trial ID. This will fail if the source trial's model architecture is
inconsistent with the model architecture of any of the trials in this experiment.

``source_checkpoint_uuid``
--------------------------

Optional. Like ``source_trial_id``, but specifies an arbitrary checkpoint from which to initialize
weights. At most one of ``source_trial_id`` or ``source_checkpoint_uuid`` should be set.

.. _experiment-configuration-searcher-adaptive:

Adaptive ASHA
=============

The ``adaptive_asha`` search method employs multiple calls to the asynchronous successive halving
algorithm (`ASHA <https://arxiv.org/pdf/1810.05934.pdf>`_) which is suitable for large-scale
experiments with hundreds or thousands of trials.

``metric``
----------

Required. The name of the validation metric used to evaluate the performance of a hyperparameter
configuration.

``time_metric``
---------------

Required. The name of the validation metric used to evaluate the progress of a given trial.

``max_time``
------------

Required. The maximum value that ``time_metric`` should take when a trial finishes training. Early
stopping is decided based on how far the ``time_metric`` has progressed towards this ``max_time``
value.

``max_trials``
--------------

Required. The number of trials, i.e., hyperparameter configurations, to evaluate.

``smaller_is_better``
---------------------

Optional. Whether to minimize or maximize the metric defined above. The default value is ``true``
(minimize).

``mode``
--------

Optional. How aggressively to perform early stopping. There are three modes: ``aggressive``,
``standard``, and ``conservative``; the default is ``standard``.

These modes differ in the degree to which early-stopping is used. In ``aggressive`` mode, the
searcher quickly stops underperforming trials, which enables the searcher to explore more
hyperparameter configurations, but at the risk of discarding a configuration too soon. On the other
end of the spectrum, ``conservative`` mode performs significantly less downsampling, but as a
consequence does not explore as many configurations given the same budget. We recommend using either
``aggressive`` or ``standard`` mode.

``divisor``
-----------

Optional. The fraction of trials to keep at each rung, and also determines the training length for
each rung. The default setting is ``4``; only advanced users should consider changing this value.

``max_rungs``
-------------

Optional. The maximum number of times we evaluate intermediate results for a trial and terminate
poorly performing trials. The default value is ``5``; only advanced users should consider changing
this value.

``max_concurrent_trials``
-------------------------

Optional. The maximum number of trials that can be worked on simultaneously. The default value is
``16``, and we set reasonable values depending on ``max_trials`` and the number of rungs in the
brackets. This is akin to controlling the degree of parallelism of the experiment. If this value is
less than the number of brackets produced by the adaptive algorithm, it will be rounded up.

``source_trial_id``
-------------------

Optional. If specified, the weights of *every* trial in the search will be initialized to the most
recent checkpoint of the given trial ID. This will fail if the source trial's model architecture is
inconsistent with the model architecture of any of the trials in this experiment.

``source_checkpoint_uuid``
--------------------------

Optional. Like ``source_trial_id``, but specifies an arbitrary checkpoint from which to initialize
weights. At most one of ``source_trial_id`` or ``source_checkpoint_uuid`` should be set.

.. _exp-config-resources:

***********
 Resources
***********

The ``resources`` section defines the resources that an experiment is allowed to use.

.. _exp-config-resources-slots-per-trial:

``slots_per_trial``
===================

Optional. The number of slots to use for each trial of this experiment. The default value is ``1``;
specifying a value greater than 1 means that multiple GPUs will be used in parallel. Training on
multiple GPUs is done using data parallelism. Configuring ``slots_per_trial`` to be greater than
``max_slots`` is not sensible and will result in an error.

.. note::

   Using ``slots_per_trial`` to enable data parallel training for PyTorch can alter the behavior of
   certain models, as described in the `PyTorch documentation
   <https://pytorch.org/docs/stable/generated/torch.nn.DataParallel.html#torch.nn.DataParallel>`__.

``slots``
=========

For historical reasons, this field usually passes config validation steps, but has no practical
effect when present in experiment config. Use :ref:`slots_per_trial
<exp-config-resources-slots-per-trial>` instead.

``max_slots``
=============

Optional. The maximum number of scheduler slots that this experiment is allowed to use at any one
time. The slot limit of an active experiment can be changed using ``det experiment set max-slots
<id> <slots>``. By default, there is no limit on the number of slots an experiment can use.

When the cluster is deployed with an :ref:`HPC workload manager <sysadmin-deploy-on-hpc>`, this
value is ignored and instead managed by the configured workload manager.

.. warning::

   ``max_slots`` is only considered when scheduling jobs; it is not currently used when provisioning
   dynamic agents. This means that we may provision more instances than the experiment can schedule.

``weight``
==========

Optional. The weight of this experiment in the scheduler. When multiple experiments are running at
the same time, the number of slots assigned to each experiment will be approximately proportional to
its weight. The weight of an active experiment can be changed using ``det experiment set weight <id>
<weight>``. The default weight is ``1``.

When the cluster is deployed with an :ref:`HPC workload manager <sysadmin-deploy-on-hpc>`, this
value is ignored and instead managed by the configured workload manager.

``shm_size``
============

Optional. The size of ``/dev/shm`` for task containers. The value can be a number in bytes or a
number with a suffix (e.g., ``128M`` for 128MiB or ``1.5G`` for 1.5GiB). Defaults to ``4294967296``
(4GiB). If set, this value overrides the value specified in the :ref:`master configuration
<master-config-reference>`.

``priority``
============

Optional. The priority assigned to this experiment. Only applicable when using the ``priority``
scheduler. Experiments with smaller priority values are scheduled before experiments with higher
priority values. If using Kubernetes, the opposite is true; experiments with higher priorities are
scheduled before those with lower priorities. Refer to :ref:`scheduling` for more information.

When the cluster is deployed with an :ref:`HPC workload manager <sysadmin-deploy-on-hpc>`, this
value is ignored and instead managed by the configured workload manager.

``resource_pool``
=================

Optional. The resource pool where this experiment will be scheduled. If no resource pool is
specified, experiments will run in the default GPU pool. Refer to :ref:`resource-pools` for more
information.

``is_single_node``
==================

Optional. When true, all the requested slots for the tasks are forced to be scheduled in a single
container on a single node, or in a single pod. When false, it may be split across different nodes
or pods. Defaults to false for experiments. This field is set to true for notebooks, tensorboards,
shells, and commands, and cannot be modified.

.. note::

   This option is currently not supported by Slurm RM.

.. _exp-resources-devices:

``devices``
===========

Optional. A list of device strings to pass to the Docker daemon. Each entry in the list is
equivalent to a ``--device DEVICE`` command-line argument to ``docker run``. ``devices`` is honored
by resource managers of type ``agent`` but is ignored by resource managers of type ``kubernetes``.
See :ref:`master configuration <master-config-reference>` for details about resource managers.

.. _exp-bind-mounts:

*************
 Bind Mounts
*************

The ``bind_mounts`` section specifies directories that are bind-mounted into every container
launched for this experiment. Bind mounts are often used to enable trial containers to access
additional data that is not part of the model definition directory.

This field should consist of an array of entries; each entry has the form described below. Users
must ensure that the specified host paths are accessible on all agent hosts (e.g., by configuring a
network file system appropriately).

``host_path``
=============

Required. The file system path on each agent to use. Must be an absolute filepath.

``container_path``
==================

Required. The file system path in the container to use. May be a relative filepath, in which case it
will be mounted relative to the working directory inside the container. It is not allowed to mount
directly into the working directory (i.e., ``container_path == "."``) to reduce the risk of
cluttering the host filesystem.

For each bind mount, the following optional fields may also be specified:

``read_only``
=============

Optional. Whether the bind-mount should be a read-only mount. Defaults to ``false``.

``propagation``
===============

Optional. `Propagation behavior
<https://docs.docker.com/engine/storage/bind-mounts/#configure-bind-propagation>`__ for replicas of
the bind-mount. Defaults to ``rprivate``.

For example, to mount ``/data`` on the host to the same path in the container, use:

.. code:: yaml

   bind_mounts:
     - host_path: /data
       container_path: /data

It is also possible to mount multiple paths:

.. code:: yaml

   bind_mounts:
     - host_path: /data
       container_path: /data
     - host_path: /shared/read-only-data
       container_path: /shared/read-only-data
       read_only: true

.. _exp-environment:

*************
 Environment
*************

The ``environment`` section defines properties of the container environment that is used to execute
workloads for this experiment. For more information on customizing the trial environment, refer to
:ref:`custom-env`.

.. _exp-environment-image:

``image``
=========

Optional. The Docker image to use when executing the workload. This image must be accessible via
``docker pull`` to every Determined agent machine in the cluster. Users can configure different
container images for NVIDIA GPU tasks using ``cuda`` key (``gpu`` prior to 0.17.6), CPU tasks using
``cpu`` key, and ROCm (AMD GPU) tasks using ``rocm`` key. Default values:

-  ``determinedai/pytorch-ngc-dev:0736b6d`` for NVIDIA GPUs and for CPUs.
-  ``determinedai/environments:rocm-5.0-pytorch-1.10-tf-2.7-rocm-0.26.4`` for ROCm.

For TensorFlow users, we provide an image that must be referenced in the experiment configuration:

-  ``determinedai/tensorflow-ngc-dev:0736b6d`` for NVIDIA GPUs and for CPUs.

When the cluster is configured with :ref:`resource_manager.type: slurm
<cluster-configuration-slurm>` and ``container_run_type: singularity``, images are executed using
the Singularity container runtime which provides additional options for specifying the container
image. The image can be:

-  A full path to a local Singularity image (beginning with a / character).

-  Any of the other supported Singularity container formats identified by prefix (e.g.
   ``instance://``, ``library://``, ``shub://``, ``oras://``, ``docker-archive://`` or
   ``docker://``). See the `Singularity run
   <https://docs.sylabs.io/guides/3.7/user-guide/cli/singularity_run.html>`__ command documentation
   for a full description of the capabilities.

-  A Singularity image provided via the ``singularity_image_root`` configured for the cluster as
   described in :ref:`slurm-image-config`.

-  If none of the above applies, Determined will apply the ``docker://`` prefix to the image.

When the cluster is configured with :ref:`resource_manager.type: slurm
<cluster-configuration-slurm>` and ``container_run_type: podman``, images are executed using the
Podman container runtime. The image can be any of the supported PodMan container formats identified
by transport (e.g. ``docker:`` (the default), ``docker-archive:``, ``docker-daemon:``, or
``oci-archive:``). Visit the `Podman
<https://docs.podman.io/en/latest/markdown/podman-run.1.html>`__ run command documentation for a
full description of the capabilities.

When the cluster is configured with :ref:`resource_manager.type: slurm
<cluster-configuration-slurm>` and ``container_run_type: enroot``, images are executed using the
Enroot container runtime. The image name must resolve to an Enroot container name created by the
user before launching the Determined task. To enable the default docker image references used by
Determined to be found in the Enroot container list the following transformations are applied to the
image name (this is the same transformation performed by the ``enroot import`` command):

-  Any forward slash character in the image name (``/``) is replaced with a plus sign (``+``)
-  Any colon (``:``) is replaced with a plus sign (``+``)

See :ref:`enroot-config-requirements` for more information.

``force_pull_image``
====================

Optional. Forcibly pull the image from the Docker registry, bypassing the Docker or Singularity
built-in cache. Defaults to ``false``.

``registry_auth``
=================

Optional. Defines the default `Docker registry credentials
<https://docs.docker.com/engine/api/v1.30/#tag/System/operation/SystemAuth>`__ to use when pulling a
custom base Docker image, if needed. Credentials are specified as the following nested fields:

-  ``username`` (required)
-  ``password`` (required)
-  ``serveraddress`` (required)
-  ``email`` (optional)

``environment_variables``
=========================

Optional. A list of environment variables that will be set in every trial container. Each element of
the list should be a string of the form ``NAME=VALUE``. See :ref:`environment-variables` for more
details. You can customize environment variables for CUDA (NVIDIA GPU), CPU, and ROCm (AMD GPU)
tasks differently by specifying a dict with ``cuda`` (``gpu`` prior to 0.17.6), ``cpu``, and
``rocm`` keys.

.. _exp-environment-pod-spec:

``pod_spec``
============

Optional. Only applicable when running Determined on Kubernetes. Applies a pod spec to the pods that
are launched by Determined for this task. See :ref:`custom-pod-specs` for details.

.. _exp-environment-add-capabilities:

``add_capabilities``
====================

Optional. A list of Linux capabilities to grant to task containers. Each entry in the list is
equivalent to a ``--cap-add CAP`` command-line argument to ``docker run``. ``add_capabilities`` is
honored by resource managers of type ``agent`` but is ignored by resource managers of type
``kubernetes``. See :ref:`master configuration <master-config-reference>` for details about resource
managers.

``drop_capabilities``
=====================

Optional. Just like ``add_capabilities`` but corresponding to the ``--cap-drop`` argument of
``docker run`` rather than ``--cap-add``.

``proxy_ports``
===============

Optional. Expose configured network ports on the chief task container. See :ref:`proxy-ports` for
details.

.. _exp-config-optimizations:

****************************
 Optimizations (deprecated)
****************************

The ``optimizations`` section contains configuration options that influence the performance of the
experiment. This section has been deprecated and should be configured in training code. Please see
:ref:`apis-howto-overview` for details specific to your training framework.

.. _config-aggregation-frequency:

``aggregation_frequency``
=========================

Optional. Specifies after how many batches gradients are exchanged during distributed training.
Defaults to ``1``.

``average_aggregated_gradients``
================================

Optional. Whether gradients accumulated across batches (when ``aggregation_frequency`` > 1) should
be divided by the ``aggregation_frequency``. Defaults to ``true``.

``average_training_metrics``
============================

Optional. For multi-GPU training, whether to average the training metrics across GPUs instead of
only using metrics from the chief GPU. This impacts the metrics shown in the Determined UI and
TensorBoard, but does not impact the outcome of training or hyperparameter search. This option is
currently supported for ``PyTorchTrial`` and ``TFKerasTrial`` instances. Defaults to ``true``.

``gradient_compression``
========================

Optional. Whether to compress gradients when they are exchanged during distributed training.
Compression may alter gradient values to achieve better space reduction. Defaults to ``false``.

``mixed_precision``
===================

Optional. Whether to use mixed precision training with PyTorch during distributed training. Setting
``O1`` enables mixed precision and loss scaling. Defaults to ``O0`` which disables mixed precision
training. This configuration setting is deprecated; users are advised to call
:meth:`context.configure_apex_amp <determined.pytorch.PyTorchTrialContext>` in the constructor of
their trial class instead.

``tensor_fusion_threshold``
===========================

Optional. The threshold in MB for batching together gradients that are exchanged during distributed
training. Defaults to ``64``.

``tensor_fusion_cycle_time``
============================

Optional. The delay (in milliseconds) between each tensor fusion during distributed training.
Defaults to ``1``.

``auto_tune_tensor_fusion``
===========================

Optional. When enabled, configures ``tensor_fusion_threshold`` and ``tensor_fusion_cycle_time``
automatically. Defaults to ``false``.

*****************
 Reproducibility
*****************

The ``reproducibility`` section specifies configuration options related to reproducible experiments.
See :ref:`reproducibility` for more details.

``experiment_seed``
===================

Optional. The random seed to use to initialize random number generators for all trials in this
experiment. Must be an integer between 0 and 2\ :sup:`31`--1. If an ``experiment_seed`` is not
explicitly specified, the master will automatically generate an experiment seed.

.. _experiment-configuration_profiling:

***********
 Profiling
***********

The ``profiling`` section specifies configuration options for the Determined system metrics
profiler. See :ref:`how-to-profiling` for a more detailed walkthrough.

``enabled``
===========

Optional. Enables system metrics profiling on the experiment, which can be viewed in the Web UI.
Defaults to false.

.. _experiment-configuration_training_units:

.. _slurm-config:

***************
 Slurm Options
***************

The ``slurm`` section specifies configuration options applicable when the cluster is configured with
:ref:`resource_manager.type: slurm <cluster-configuration-slurm>`.

``gpu_type``
============

Optional. An optional GPU type name to be included in the generated Slurm ``--gpus`` or ``--gres``
option if you have configured GPU types within your Slurm gres configuration. Specify this option to
select that specific GPU type when there are multiple GPU types within the Slurm partition. The
default is to select GPUs without regard to their type. For example, you can request the ``tesla``
GPU type with:

.. code:: yaml

   slurm:
      gpu_type: tesla

.. _sbatch-args:

``sbatch_args``
===============

Optional. Additional Slurm options to be passed when launching trials with ``sbatch``. These options
enable control of Slurm options not otherwise managed by Determined. For example, to specify
required memory per CPU and exclusive access to an entire node when scheduled, you could specify:

.. code:: yaml

   slurm:
      sbatch_args:
         - --mem-per-cpu=10
         - --exclusive

.. _slots-per-node:

``slots_per_node``
==================

Optional. The minimum number of slots required for a node to be scheduled during a trial. If
:ref:`gres_supported <cluster-configuration-slurm>` is false, specify ``slots_per_node`` in order to
utilize more than one GPU per node. It is the user’s responsibility to ensure that
``slots_per_node`` GPUs will be available on nodes selected for the job using other configurations
such as targeting a specific resource pool with only GPU nodes or specifying a Slurm constraint in
the experiment configuration.

.. _pbs-config:

*************
 PBS Options
*************

The ``pbs`` section specifies configuration options applicable when the cluster is configured with
:ref:`resource_manager.type: pbs <cluster-configuration-slurm>`.

.. _pbsbatch-args:

``pbsbatch_args``
=================

Optional. Additional PBS options to be passed when launching trials with ``qsub``. These options
enable control of PBS options not otherwise managed by Determined. For example, to specify that the
job should have a priority of ``1000`` and a project name of ``MyProjectName``, you could specify:

.. code:: yaml

   pbs:
      pbsbatch_args:
         - -p1000
         - -PMyProjectName

Requesting of resources and job placement may be influenced through use of ``-l``, however chunk
count, chunk arrangement, and GPU or CPU counts per chunk (depending on the value of ``slot_type``)
are controlled by Determined; any values specified for these quantities will be ignored. Consider if
the following were specified for a CUDA experiment:

.. code:: yaml

   pbs:
      pbsbatch_args:
         - -l select=2:ngpus=4:mem=4gb
         - -l place=scatter:shared
         - -l walltime=1:00:00

The chunk count (two), the GPU count per chunk (four), and the chunk arrangement (scatter) will all
be ignored in favor of values calculated by Determined.

``slots_per_node``
==================

Optional. Specifies the minimum number of slots required for a node to be scheduled during a trial.
If :ref:`gres_supported <cluster-configuration-slurm>` is set to ``false``, specify
``slots_per_node`` in order to utilize more than one GPU per node. It is the user’s responsibility
to ensure that ``slots_per_node`` GPUs will be available on the nodes selected for the job using
other configurations such as targeting a specific resource pool with only ``slots_per_node`` GPU
nodes or specifying a PBS constraint in the experiment configuration.
