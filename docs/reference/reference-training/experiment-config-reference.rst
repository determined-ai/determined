.. _experiment-config-reference:

.. _experiment-configuration:

####################################
 Experiment Configuration Reference
####################################

The behavior of an experiment can be configured via a YAML file. A configuration file is typically
passed as a command-line argument when an experiment is created with the Determined CLI. For
example:

.. code::

   det experiment create config-file.yaml model-directory

**********
 Metadata
**********

**Optional Fields**

``name``
   A short human-readable name for the experiment.

``description``
   A human-readable description of the experiment. This does not need to be unique but should be
   limited to less than 255 characters for the best experience.

``labels``
   A list of label names (strings). Assigning labels to experiments allows you to identify
   experiments that share the same property or should be grouped together. You can add and remove
   labels using either the CLI (``det experiment label``) or the WebUI.

.. _experiment-config-data:

``data``
   This field can be used to specify information about how the experiment accesses and loads
   training data. The content and format of this field is user-defined: it should be used to specify
   whatever configuration is needed for loading data for use by the experiment's model definition.
   For example, if your experiment loads data from Amazon S3, the ``data`` field might contain the
   S3 bucket name, object prefix, and AWS authentication credentials.

``workspace``
   The name of the pre-existing workspace where you want to create the experiment. The ``workspace``
   and ``project`` fields must either both be present or both be absent. If they are absent, the
   experiment is placed in the ``Uncategorized`` project in the ``Uncategorized`` workspace. You can
   manage workspaces using the CLI ``det workspace help`` command or the WebUI.

``project``
   The name of the pre-existing project inside ``workspace`` where you want to create the
   experiment. The ``workspace`` and ``project`` fields must either both be present or both be
   absent. If they are absent, the experiment is placed in the ``Uncategorized`` project in the
   ``Uncategorized`` workspace. You can manage projects using the CLI ``det project help`` command
   or the WebUI.

************
 Entrypoint
************

**Required Fields**

.. _experiment-config-entrypoint:

``entrypoint``

A model definition trial class specification or Python launcher script, which is the model
processing entrypoint. This field can have the following formats.

Formats that specify a trial class have the form ``<module>:<object_reference>``.

The ``<module>`` field specifies the module containing the trial class in the model definition,
relative to root.

The ``<object_reference>`` specifies the trial class name in the module, which can be a nested
object delimited by a period (``.``).

Examples:

-  ``:MnistTrial`` expects an *MnistTrial* class exposed in a ``__init__.py`` file at the top level
   of the context directory.
-  ``model_def:CIFAR10Trial`` expects a *CIFAR10Trial* class defined in the ``model_def.py`` file at
   the top level of the context directory.
-  ``determined_lib.trial:trial_classes.NestedTrial`` expects a ``NestedTrial`` class, which is an
   attirbute of ``trial_classes`` defined in the ``determined_lib/trial.py`` file.

These formats follow Python `Entry points
<https://packaging.python.org/specifications/entry-points/>`_ specification except that the context
directory name is prefixed by ``<module>`` or used as the module if the ``<module>`` field is empty.

Arbitrary Script
================

An arbitrary entrypoint script name.

Example:

.. code:: yaml

   entrypoint: ./hello.sh

Preconfigured Launch Module with Script
=======================================

The name of a preconfigured launch module and script name.

Example:

.. code:: yaml

   entrypoint: python3 -m (LAUNCH_MODULE) train.py

``LAUNCH_MODULE`` options:

-  Horovod (determined.launch.horovod)
-  PyTorch (determined.launch.torch_distributed)
-  Deepspeed (determined.launch.deepspeed)

Preconfigured Launch Module with Legacy Trial Definition
========================================================

The name of a preconfigured launch module and legacy trial class specification.

Example:

.. code:: yaml

   entrypoint: python3 -m (LAUNCH_MODULE) --trial model_def:Trial

``LAUNCH_MODULE`` options: [need literals for these]

-  Horovod (determined.launch.horovod)
-  PyTorch (determined.launch.torch_distributed)
-  Deepspeed (determined.launch.deepspeed)

Legacy Trial Definition
=======================

A legacy trial class specification.

Example:

.. code:: yaml

   entrypoint: model_def:Trial

*****************
 Basic Behaviors
*****************

**Optional Fields**

.. _scheduling-unit:

``scheduling_unit``
   Instructs how frequent to perform system operations, such as periodic checkpointing and
   preemption, in the unit of batches. The number of records in a batch is controlled by the
   :ref:`global_batch_size <config-global-batch-size>` hyperparameter. Defaults to ``100``.

   -  Setting this value too small can increase the overhead of system operations and decrease
      training throughput.
   -  Setting this value too large might prevent the system from reallocating resources from this
      workload to another, potentially more important, workload.
   -  As a rule of thumb, it should be set to the number of batches that can be trained in roughly
      60--180 seconds.

.. _config-records-per-epoch:

``records_per_epoch``
   The number of records in the training data set. It must be configured if you want to specify
   ``min_validation_period``, ``min_checkpoint_period``, and ``searcher.max_length`` in units of
   ``epochs``.

   -  The system does not attempt to determine the size of an epoch automatically, because the size
      of the training set might vary based on data augmentation, changes to external storage, or
      other factors.

.. _max-restarts:

``max_restarts``
   The maximum number of times that trials in this experiment will be restarted due to an error. If
   an error occurs while a trial is running (e.g., a container crashes abruptly), the Determined
   master will automatically restart the trial and continue running it. This parameter specifies a
   limit on the number of times to try restarting a trial; this ensures that Determined does not go
   into an infinite loop if a trial encounters the same error repeatedly. Once ``max_restarts``
   trial failures have occurred for a given experiment, subsequent failed trials will not be
   restarted -- instead, they will be marked as errored. The experiment itself will continue
   running; an experiment is considered to complete successfully if at least one of its trials
   completes successfully. The default value is ``5``.

*******************
 Validation Policy
*******************

**Optional Fields**

.. _experiment-config-min-validation-period:

``min_validation_period``
   Instructs the minimum frequency for running validation for each trial.

   -  This needs to be set in the unit of records, batches, or epochs using a nested dictionary. For
      example:

      .. code:: yaml

         min_validation_period:
            epochs: 2

   -  If this is in the unit of epochs, :ref:`records_per_epoch <config-records-per-epoch>` must be
      specified.

.. _experiment-config-perform-initial-validation:

``perform_initial_validation``
   Instructs Determined to perform an initial validation before any training begins, for each trial.
   This can be useful to determine a baseline when fine-tuning a model on a new dataset.

*******************
 Checkpoint Policy
*******************

We will checkpoint in the following situations:

#. During training, periodically to keep record of the training progress;
#. During training, to allow the trial's execution to be recovered from resuming or errors;
#. When the trial is completed;
#. Before the searcher makes a decision based on the validation of trials, to maintain consistency
   in the event of a failure.

**Optional Fields**

.. _experiment-config-min-checkpoint-period:

``min_checkpoint_period``
   Instructs the minimum frequency for running checkpointing for each trial.

   -  This needs to be set in the unit of records, batches, or epochs using a nested dictionary. For
      example:

      .. code:: yaml

         min_checkpoint_period:
            epochs: 2

   -  If this is in the unit of epochs, :ref:`records_per_epoch <config-records-per-epoch>` must be
      specified.

``checkpoint_policy``
   Controls how Determined performs checkpoints after validation operations, if at all. Should be
   set to one of the following values:

   -  ``best`` (default): A checkpoint will be taken after every validation operation that performs
      better than all previous validations for this experiment. Validation metrics are compared
      according to the ``metric`` and ``smaller_is_better`` options in the :ref:`searcher
      configuration <experiment-configuration_searcher>`.

   -  ``all``: A checkpoint will be taken after every validation, no matter the validation
      performance.

   -  ``none``: A checkpoint will never be taken *due* to a validation. However, even with this
      policy selected, checkpoints are still expected to be taken after the trial is finished
      training, due to cluster scheduling decisions, before search method decisions, or due to
      :ref:`min_checkpoint_period <experiment-config-min-checkpoint-period>`.

.. _checkpoint-storage:

********************
 Checkpoint Storage
********************

The ``checkpoint_storage`` section defines how model checkpoints will be stored. A checkpoint
contains the architecture and weights of the model being trained. Each checkpoint has a UUID, which
is used as the name of the checkpoint directory on the external storage system.

If this field is not specified, the experiment will default to the checkpoint storage configured in
the :ref:`master-config-reference`.

.. _checkpoint-garbage-collection:

Checkpoint Garbage Collection
=============================

When an experiment finishes, the system will optionally delete some checkpoints to reclaim space.
The ``save_experiment_best``, ``save_trial_best`` and ``save_trial_latest`` parameters specify which
checkpoints to save. If multiple ``save_*`` parameters are specified, the union of the specified
checkpoints are saved.

-  ``save_experiment_best``: The number of the best checkpoints with validations over all trials to
   save (where best is measured by the validation metric specified in the searcher configuration).
-  ``save_trial_best``: The number of the best checkpoints with validations of each trial to save.
-  ``save_trial_latest``: The number of the latest checkpoints of each trial to save.

These fields default to the following respective value:

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

Storage Type
============

Determined currently supports several kinds of checkpoint storage, ``gcs``, ``hdfs``, ``s3``,
``azure``, and ``shared_fs``, identified by the ``type`` subfield. Additional fields may also be
required, depending on the type of checkpoint storage in use. For example, to store checkpoints on
Google Cloud Storage:

.. code:: yaml

   checkpoint_storage:
     type: gcs
     bucket: <your-bucket-name>

Google Cloud Storage
--------------------

If ``type: gcs`` is specified, checkpoints will be stored on Google Cloud Storage (GCS).
Authentication is done using GCP's "`Application Default Credentials
<https://googleapis.dev/python/google-api-core/latest/auth.html>`__" approach. When using Determined
inside Google Compute Engine (GCE), the simplest approach is to ensure that the VMs used by
Determined are running in a service account that has the "Storage Object Admin" role on the GCS
bucket being used for checkpoints. As an alternative (or when running outside of GCE), you can add
the appropriate `service account credentials
<https://cloud.google.com/docs/authentication/production#obtaining_and_providing_service_account_credentials_manually>`__
to your container (e.g., via a bind-mount), and then set the ``GOOGLE_APPLICATION_CREDENTIALS``
environment variable to the container path where the credentials are located. See
:ref:`environment-variables` for more details on how to set environment variables in containers.

**Required Fields**

``bucket``
   The GCS bucket name to use.

**Optional Fields**

``prefix``
   The optional path prefix to use. Must not contain ``..``. Note: Prefix is normalized, e.g.,
   ``/pre/.//fix`` -> ``/pre/fix``

HDFS
----

If ``type: hdfs`` is specified, checkpoints will be stored in HDFS using the `WebHDFS
<http://hadoop.apache.org/docs/current/hadoop-project-dist/hadoop-hdfs/WebHDFS.html>`__ API for
reading and writing checkpoint resources.

**Required Fields**

``hdfs_url``
   Hostname or IP address of HDFS namenode, prefixed with protocol, followed by WebHDFS port on
   namenode. Multiple namenodes are allowed as a semicolon-separated list (e.g.,
   ``"http://namenode1:50070;http://namenode2:50070"``).

``hdfs_path``
   The prefix path where all checkpoints will be written to and read from. The resources of each
   checkpoint will be saved in a subdirectory of ``hdfs_path``, where the subdirectory name is the
   checkpoint's UUID.

**Optional Fields**

``user``
   The user name to use for all read and write requests. If not specified, this defaults to the user
   of the trial runner container.

Amazon S3
---------

If ``type: s3`` is specified, checkpoints will be stored in Amazon S3 or an S3-compatible object
store such as `MinIO <https://min.io/>`__.

**Required Fields**

``bucket``
   The S3 bucket name to use.

``access_key``
   The AWS access key to use.

``secret_key``
   The AWS secret key to use.

**Optional Fields**

``prefix``
   The optional path prefix to use. Must not contain ``..``. Note: Prefix is normalized, e.g.,
   ``/pre/.//fix`` -> ``/pre/fix``

``endpoint_url``
   The endpoint to use for S3 clones, e.g., ``http://127.0.0.1:8080/``. If not specified, Amazon S3
   will be used.

Azure Blob Storage
------------------

If ``type: azure`` is specified, checkpoints will be stored in Microsoft's Azure Blob Storage.

Please only specify one of ``connection_string`` or the ``account_url``, ``credential`` tuple.

**Required Fields**

``container``
   The Azure Blob Storage container name to use.

``connection_string``
   The connection string for the Azure Blob Storage service account to use.

``account_url``
   The account URL for the Azure Blob Storage service account to use.

**Optional Fields**

``credential``
   The credential to use with the ``account_url``.

Shared File System
------------------

If ``type: shared_fs`` is specified, checkpoints will be written to a directory on the agent's file
system. The assumption is that the system administrator has arranged for the same directory to be
mounted at every agent machine, and for the content of this directory to be the same on all agent
hosts (e.g., by using a distributed or network file system such as `GlusterFS
<https://www.gluster.org/>`__ or `NFS <https://en.wikipedia.org/wiki/Network_File_System>`__).

.. warning::

   When downloading checkpoints from a shared file system (e.g., using ``det checkpoint download``),
   we assume the same shared file system is mounted locally at the same ``host_path``.

**Required Fields**

``host_path``
   The file system path on each agent to use. This directory will be mounted to
   ``/determined_shared_fs`` inside the trial container.

**Optional Fields**

``storage_path``
   The path where checkpoints will be written to and read from. Must be a subdirectory of the
   ``host_path`` or an absolute path containing the ``host_path``. If not specified, checkpoints are
   written to and read from the ``host_path``.

``propagation``
   `Propagation behavior
   <https://docs.docker.com/storage/bind-mounts/#configure-bind-propagation>`__ for replicas of the
   bind-mount. Defaults to ``rprivate``.

.. _experiment-configuration_hyperparameters:

*****************
 Hyperparameters
*****************

The ``hyperparameters`` section defines the hyperparameter space for the experiment. Which
hyperparameters are appropriate for a given model is up to the user and depends on the nature of the
model being trained. In Determined, it is common to specify hyperparameters that influence many
aspects of the model's behavior, including how data augmentation is done, the architecture of the
neural network, and which optimizer to use, along with how that optimizer should be configured.

The value chosen for a hyperparameter in a given trial can be accessed via the trial context using
:func:`context.get_hparam() <determined.TrialContext.get_hparam>`. For instance, the current value
of a hyperparameter named ``learning_rate`` can be accessed by
``context.get_hparam("learning_rate")``.

.. _config-global-batch-size:

.. note::

   Every experiment must specify a hyperparameter named ``global_batch_size``. This is because this
   hyperparameter is treated specially: when doing distributed training, the global batch size must
   be known so that the per-worker batch size can be computed appropriately. Batch size per slot is
   computed at runtime, based on the number of slots used to train a single trial of this experiment
   (see :ref:`resources.slots_per_trial <exp-config-resources-slots-per-trial>`). The updated values
   should be accessed via the trial context, using :func:`context.get_per_slot_batch_size()
   <determined.TrialContext.get_per_slot_batch_size>` and :func:`context.get_global_batch_size()
   <determined.TrialContext.get_global_batch_size>`.

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
different hyperparameter search algorithms: ``adaptive_asha``, ``random``, and ``grid``. To define
your own hyperparameter search algorithm, specify the ``custom`` searcher. For more information
about custom search algorithms, see :ref:`topic-guides_hp-tuning-det_custom`.

The name of the hyperparameter search algorithm to use is configured via the ``name`` field; the
remaining fields configure the behavior of the searcher and depend on the searcher being used. For
example, to configure a ``random`` hyperparameter search that trains 5 trials for 1000 batches each:

.. code:: yaml

   searcher:
     name: random
     metric: accuracy
     max_trials: 5
     max_length:
       batches: 1000

For details on using Determined to perform hyperparameter search, refer to
:ref:`hyperparameter-tuning`. For more information on the search methods supported by Determined,
refer to :ref:`hyperparameter-tuning`.

Single
======

The ``single`` search method does not perform a hyperparameter search at all; rather, it trains a
single trial for a fixed length. When using this search method, all of the hyperparameters specified
in the :ref:`hyperparameters <experiment-configuration_hyperparameters>` section must be constants.
By default, validation metrics are only computed once, after the specified length of training has
been completed; :ref:`min_validation_period <experiment-config-min-validation-period>` can be used
to specify that validation metrics should be computed more frequently.

**Required Fields**

``metric``
   The name of the validation metric used to evaluate the performance of a hyperparameter
   configuration.

.. _experiment-configuration_single-searcher-max-length:

``max_length``
   The length of the trial.

   -  This needs to be set in the unit of records, batches, or epochs using a nested dictionary. For
      example:

      .. code:: yaml

         max_length:
            epochs: 2

   -  If this is in the unit of epochs, :ref:`records_per_epoch <config-records-per-epoch>` must be
      specified.

**Optional Fields**

``smaller_is_better``
   Whether to minimize or maximize the metric defined above. The default value is ``true``
   (minimize).

``source_trial_id``
   If specified, the weights of this trial will be initialized to the most recent checkpoint of the
   given trial ID. This will fail if the source trial's model architecture is inconsistent with the
   model architecture of this experiment.

``source_checkpoint_uuid``
   Like ``source_trial_id``, but specifies an arbitrary checkpoint from which to initialize weights.
   At most one of ``source_trial_id`` or ``source_checkpoint_uuid`` should be set.

Random
======

The ``random`` search method implements a simple random search. The user specifies how many
hyperparameter configurations should be trained and how long each configuration should be trained
for; the configurations are sampled randomly from the hyperparameter space. Each trial is trained
for the specified length and then validation metrics are computed. :ref:`min_validation_period
<experiment-config-min-validation-period>` can be used to specify that validation metrics should be
computed more frequently.

**Required Fields**

``metric``
   The name of the validation metric used to evaluate the performance of a hyperparameter
   configuration.

``max_trials``
   The number of trials, i.e., hyperparameter configurations, to evaluate.

``max_length``
   The length of each trial.

   -  This needs to be set in the unit of records, batches, or epochs using a nested dictionary. For
      example:

      .. code:: yaml

         max_length:
            epochs: 2

   -  If this is in the unit of epochs, :ref:`records_per_epoch <config-records-per-epoch>` must be
      specified.

**Optional Fields**

``smaller_is_better``
   Whether to minimize or maximize the metric defined above. The default value is ``true``
   (minimize).

``max_concurrent_trials``
   The maximum number of trials that can be worked on simultaneously. The default value is ``16``.
   When the value is ``0`` we will work on as many trials as possible.

``source_trial_id``
   If specified, the weights of *every* trial in the search will be initialized to the most recent
   checkpoint of the given trial ID. This will fail if the source trial's model architecture is
   incompatible with the model architecture of any of the trials in this experiment.

``source_checkpoint_uuid``
   Like ``source_trial_id`` but specifies an arbitrary checkpoint from which to initialize weights.
   At most one of ``source_trial_id`` or ``source_checkpoint_uuid`` should be set.

Grid
====

The ``grid`` search method performs a grid search. The coordinates of the hyperparameter grid are
specified via the ``hyperparameters`` field. For more details see the
:ref:`topic-guides_hp-tuning-det_grid`.

**Required Fields**

``metric``
   The name of the validation metric used to evaluate the performance of a hyperparameter
   configuration.

``max_length``
   The length of each trial.

   -  This needs to be set in the unit of records, batches, or epochs using a nested dictionary. For
      example:

      .. code:: yaml

         max_length:
            epochs: 2

   -  If this is in the unit of epochs, :ref:`records_per_epoch <config-records-per-epoch>` must be
      specified.

**Optional Fields**

``smaller_is_better``
   Whether to minimize or maximize the metric defined above. The default value is ``true``
   (minimize).

``max_concurrent_trials``
   The maximum number of trials that can be worked on simultaneously. The default value is ``16``.
   When the value is ``0`` we will work on as many trials as possible.

``source_trial_id``
   If specified, the weights of this trial will be initialized to the most recent checkpoint of the
   given trial ID. This will fail if the source trial's model architecture is inconsistent with the
   model architecture of this experiment.

``source_checkpoint_uuid``
   Like ``source_trial_id``, but specifies an arbitrary checkpoint from which to initialize weights.
   At most one of ``source_trial_id`` or ``source_checkpoint_uuid`` should be set.

.. _experiment-configuration-searcher-adaptive:

Adaptive ASHA
=============

The ``adaptive_asha`` search method employs multiple calls to the asynchronous successive halving
algorithm (`ASHA <https://arxiv.org/pdf/1810.05934.pdf>`_) which is suitable for large-scale
experiments with hundreds or thousands of trials.

**Required Fields**

``metric``
   The name of the validation metric used to evaluate the performance of a hyperparameter
   configuration.

``max_length``
   The maximum training length of any one trial. The vast majority of trials will be stopped early,
   and thus only a small fraction of trials will actually be trained for this long. This quantity is
   domain-specific and should roughly reflect the length of training needed for the model to
   converge on the data set.

   -  This needs to be set in the unit of records, batches, or epochs using a nested dictionary. For
      example:

      .. code:: yaml

         max_length:
            epochs: 2

   -  If this is in the unit of epochs, :ref:`records_per_epoch <config-records-per-epoch>` must be
      specified.

``max_trials``
   The number of trials, i.e., hyperparameter configurations, to evaluate.

**Optional Fields**

``smaller_is_better``
   Whether to minimize or maximize the metric defined above. The default value is ``true``
   (minimize).

``mode``
   How aggressively to perform early stopping. There are three modes: ``aggressive``, ``standard``,
   and ``conservative``; the default is ``standard``.

   These modes differ in the degree to which early-stopping is used. In ``aggressive`` mode, the
   searcher quickly stops underperforming trials, which enables the searcher to explore more
   hyperparameter configurations, but at the risk of discarding a configuration too soon. On the
   other end of the spectrum, ``conservative`` mode performs significantly less downsampling, but as
   a consequence does not explore as many configurations given the same budget. We recommend using
   either ``aggressive`` or ``standard`` mode.

``stop_once``
   If ``stop_once`` is set to ``true``, we will use a variant of ASHA that will not resume trials
   once stopped. This variant defaults to continuing training and will only stop trials if there is
   enough evidence to terminate training. We recommend using this version of ASHA when training a
   trial for the max length as fast as possible is important or when fault tolerance is too
   expensive.

``divisor``
   The fraction of trials to keep at each rung, and also determines the training length for each
   rung. The default setting is ``4``; only advanced users should consider changing this value.

``max_rungs``
   The maximum number of times we evaluate intermediate results for a trial and terminate poorly
   performing trials. The default value is ``5``; only advanced users should consider changing this
   value.

``max_concurrent_trials``
   The maximum number of trials that can be worked on simultaneously. The default value is ``16``,
   and we set reasonable values depending on ``max_trials`` and the number of rungs in the brackets.
   This is akin to controlling the degree of parallelism of the experiment. If this value is less
   than the number of brackets produced by the adaptive algorithm, it will be rounded up.

``source_trial_id``
   If specified, the weights of *every* trial in the search will be initialized to the most recent
   checkpoint of the given trial ID. This will fail if the source trial's model architecture is
   inconsistent with the model architecture of any of the trials in this experiment.

``source_checkpoint_uuid``
   Like ``source_trial_id``, but specifies an arbitrary checkpoint from which to initialize weights.
   At most one of ``source_trial_id`` or ``source_checkpoint_uuid`` should be set.

.. _exp-config-resources:

***********
 Resources
***********

The ``resources`` section defines the resources that an experiment is allowed to use.

**Optional Fields**

.. _exp-config-resources-slots-per-trial:

``slots_per_trial``
   The number of slots to use for each trial of this experiment. The default value is ``1``;
   specifying a value greater than 1 means that multiple GPUs will be used in parallel. Training on
   multiple GPUs is done using data parallelism. Configuring ``slots_per_trial`` to be greater than
   ``max_slots`` is not sensible and will result in an error.

   .. note::

      Using ``slots_per_trial`` to enable data parallel training for PyTorch can alter the behavior
      of certain models, as described in the `PyTorch documentation
      <https://pytorch.org/docs/stable/generated/torch.nn.DataParallel.html#torch.nn.DataParallel>`__.

``max_slots``
   The maximum number of scheduler slots that this experiment is allowed to use at any one time. The
   slot limit of an active experiment can be changed using ``det experiment set max-slots <id>
   <slots>``. By default, there is no limit on the number of slots an experiment can use.

   When the cluster is deployed with an :ref:`HPC workload manager <sysadmin-deploy-on-hpc>`, this
   value is ignored and instead managed by the configured workload manager.

   .. warning::

      ``max_slots`` is only considered when scheduling jobs; it is not currently used when
      provisioning dynamic agents. This means that we may provision more instances than the
      experiment can schedule.

``weight``
   The weight of this experiment in the scheduler. When multiple experiments are running at the same
   time, the number of slots assigned to each experiment will be approximately proportional to its
   weight. The weight of an active experiment can be changed using ``det experiment set weight <id>
   <weight>``. The default weight is ``1``.

   When the cluster is deployed with an :ref:`HPC workload manager <sysadmin-deploy-on-hpc>`, this
   value is ignored and instead managed by the configured workload manager.

``shm_size``
   The size of ``/dev/shm`` for task containers. The value can be a number in bytes or a number with
   a suffix (e.g., ``128M`` for 128MiB or ``1.5G`` for 1.5GiB). Defaults to ``4294967296`` (4GiB).
   If set, this value overrides the value specified in the :ref:`master configuration
   <master-config-reference>`.

``priority``
   The priority assigned to this experiment. Only applicable when using the ``priority`` scheduler.
   Experiments with smaller priority values are scheduled before experiments with higher priority
   values. If using Kubernetes, the opposite is true; experiments with higher priorities are
   scheduled before those with lower priorities. Refer to :ref:`scheduling` for more information.

   When the cluster is deployed with an :ref:`HPC workload manager <sysadmin-deploy-on-hpc>`, this
   value is ignored and instead managed by the configured workload manager.

``resource_pool``
   The resource pool where this experiment will be scheduled. If no resource pool is specified,
   experiments will run in the default GPU pool. Refer to :ref:`resource-pools` for more
   information.

.. _exp-resources-devices:

``devices``
   A list of device strings to pass to the Docker daemon. Each entry in the list is equivalent to a
   ``--device DEVICE`` command-line argument to ``docker run``. ``devices`` is honored by resource
   managers of type ``agent`` but is ignored by resource managers of type ``kubernetes``. See
   :ref:`master configuration <master-config-reference>` for details about resource managers.

``agent_label``
   This field has been deprecated and will be ignored. Use ``resource_pool`` instead.

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

For each bind mount, the following fields are required:

``host_path``
   The file system path on each agent to use. Must be an absolute filepath.

``container_path``
   The file system path in the container to use. May be a relative filepath, in which case it will
   be mounted relative to the working directory inside the container. It is not allowed to mount
   directly into the working directory (i.e., ``container_path == "."``) to reduce the risk of
   cluttering the host filesystem.

For each bind mount, the following optional fields may also be specified:

``read_only``
   Whether the bind-mount should be a read-only mount. Defaults to ``false``.

``propagation``
   `Propagation behavior
   <https://docs.docker.com/storage/bind-mounts/#configure-bind-propagation>`__ for replicas of the
   bind-mount. Defaults to ``rprivate``.

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

**Optional Fields**

.. _exp-environment-image:

``image``
   The Docker image to use when executing the workload. This image must be accessible via ``docker
   pull`` to every Determined agent machine in the cluster. Users can configure different container
   images for NVIDIA GPU tasks using ``cuda`` key (``gpu`` prior to 0.17.6), CPU tasks using ``cpu``
   key, and ROCm (AMD GPU) tasks using ``rocm`` key. Default values:

   -  ``determinedai/environments-dev:cuda-11.3-pytorch-1.12-tf-2.8-gpu-0.21.2`` for NVIDIA GPUs.
   -  ``determinedai/environments-dev:py-3.8-pytorch-1.12-tf-2.8-cpu-0.21.2`` for CPUs.
   -  ``determinedai/environments-dev:rocm-5.0-pytorch-1.10-tf-2.7-rocm-0.21.2`` for ROCm.

   When the cluster is configured with :ref:`resource_manager.type: slurm
   <cluster-configuration-slurm>` and ``container_run_type: singularity``, images are executed using
   the Singularity container runtime which provides additional options for specifying the container
   image. The image can be:

      -  A full path to a local Singulary image (beginning with a / character).

      -  Any of the other supported Singularity container formats identified by prefix (e.g.
         ``instance://``, ``library://``, ``shub://``, ``oras://``, or ``docker://``). See the
         `Singularity run <https://docs.sylabs.io/guides/3.7/user-guide/cli/singularity_run.html>`__
         command documentation for a full description of the capabilities.

      -  A Singularity image provided via the `singularity_image_root` configured for the cluster as
         described in :ref:`slurm-image-config`.

      -  If none of the above applies, Determined will apply the ``docker://`` prefix to the image.

   When the cluster is configured with :ref:`resource_manager.type: slurm
   <cluster-configuration-slurm>` and ``container_run_type: podman``, images are executed using the
   PodMan container runtime. The image can be any of the supported PodMan container formats
   identified by transport (e.g. ``docker:`` (the default), ``docker-archive:``, ``docker-daemon:``,
   or ``oci-archive:``). See the `PodMan run
   <https://docs.podman.io/en/latest/markdown/podman-run.1.html>`__ command documentation for a full
   description of the capabilities.

   When the cluster is configured with :ref:`resource_manager.type: slurm
   <cluster-configuration-slurm>` and ``container_run_type: enroot``, images are executed using the
   Enroot container runtime. The image name must resolve to an Enroot container name created by the
   user before launching the Determined task. To enable the default docker image references used by
   Determined to be found in the Enroot container list the following transformations are applied to
   the image name (this is the same transformation performed by the ``enroot import`` command):

      -  Any forward slash character in the image name (``/``) is replaced with a plus sign (``+``)
      -  Any colon (``:``) is replaced with a plus sign (``+``)

   See :ref:`enroot-config-requirements` for more information.

``force_pull_image``
   Forcibly pull the image from the Docker registry, bypassing the Docker or Singularity built-in
   cache. Defaults to ``false``.

``registry_auth``
   The `Docker registry credentials
   <https://docs.docker.com/engine/api/v1.30/#operation/SystemAuth>`__ to use when pulling a custom
   base Docker image, if needed. Credentials are specified as the following nested fields:

   -  ``username`` (required)
   -  ``password`` (required)
   -  ``serveraddress`` (required)
   -  ``email`` (optional)

``environment_variables``
   A list of environment variables that will be set in every trial container. Each element of the
   list should be a string of the form ``NAME=VALUE``. See :ref:`environment-variables` for more
   details. Users can customize environment variables for CUDA (NVIDIA GPU), CPU, and ROCm (AMD GPU)
   tasks differently by specifying a dict with ``cuda`` (``gpu`` prior to 0.17.6), ``cpu``, and
   ``rocm`` keys.

.. _exp-environment-pod-spec:

``pod_spec``
   Only applicable when running Determined on Kubernetes. Applies a pod spec to the pods that are
   launched by Determined for this task. See :ref:`custom-pod-specs` for details.

.. _exp-environment-add-capapbilities:

``add_capabilities``
   A list of Linux capabilities to grant to task containers. Each entry in the list is equivalent to
   a ``--cap-add CAP`` command-line argument to ``docker run``. ``add_capabilities`` is honored by
   resource managers of type ``agent`` but is ignored by resource managers of type ``kubernetes``.
   See :ref:`master configuration <master-config-reference>` for details about resource managers.

``drop_capabilities``
   Just like ``add_capabilities`` but corresponding to the ``--cap-drop`` argument of ``docker run``
   rather than ``--cap-add``.

``proxy_ports``: Expose configured network ports on the chief task container. See :ref:`proxy-ports`
for details.

***************
 Optimizations
***************

The ``optimizations`` section contains configuration options that influence the performance of the
experiment.

**Optional Fields**

.. _config-aggregation-frequency:

``aggregation_frequency``
   Specifies after how many batches gradients are exchanged during :ref:`multi-gpu-training`.
   Defaults to ``1``.

``average_aggregated_gradients``
   Whether gradients accumulated across batches (when ``aggregation_frequency`` > 1) should be
   divided by the ``aggregation_frequency``. Defaults to ``true``.

``average_training_metrics``
   For multi-GPU training, whether to average the training metrics across GPUs instead of only using
   metrics from the chief GPU. This impacts the metrics shown in the Determined UI and TensorBoard,
   but does not impact the outcome of training or hyperparameter search. This option is currently
   supported for ``PyTorchTrial`` and ``TFKerasTrial`` instances. Defaults to ``true``.

``gradient_compression``
   Whether to compress gradients when they are exchanged during :ref:`multi-gpu-training`.
   Compression may alter gradient values to achieve better space reduction. Defaults to ``false``.

``mixed_precision``
   Whether to use mixed precision training with PyTorch during :ref:`multi-gpu-training`. Setting
   ``O1`` enables mixed precision and loss scaling. Defaults to ``O0`` which disables mixed
   precision training. This configuration setting is deprecated; users are advised to call
   :meth:`context.configure_apex_amp <determined.pytorch.PyTorchTrialContext>` in the constructor of
   their trial class instead.

``tensor_fusion_threshold``
   The threshold in MB for batching together gradients that are exchanged during
   :ref:`multi-gpu-training`. Defaults to ``64``.

``tensor_fusion_cycle_time``
   The delay (in milliseconds) between each tensor fusion during :ref:`multi-gpu-training`. Defaults
   to ``5``.

``auto_tune_tensor_fusion``
   When enabled, configures ``tensor_fusion_threshold`` and ``tensor_fusion_cycle_time``
   automatically. Defaults to ``false``.

*****************
 Reproducibility
*****************

The ``reproducibility`` section specifies configuration options related to reproducible experiments.
See :ref:`reproducibility` for more details.

**Optional Fields**

``experiment_seed``
   The random seed to use to initialize random number generators for all trials in this experiment.
   Must be an integer between 0 and 2\ :sup:`31`--1. If an ``experiment_seed`` is not explicitly
   specified, the master will automatically generate an experiment seed.

.. _experiment-configuration_profiling:

***********
 Profiling
***********

The ``profiling`` section specifies configuration options related to profiling experiments. See
:ref:`how-to-profiling` for a more detailed walkthrough.

**Optional Fields**

``profiling``
   Profiling is supported for all frameworks, though timings are only collected for
   ``PyTorchTrial``. Profiles are collected for a maximum of 5 minutes, regardless of the settings
   below.

   ``enabled``
      Defines whether profiles should be collected or not. Defaults to false.

   ``begin_on_batch``
      Specifies the batch on which profiling should begin.

   ``end_after_batch``
      Specifies the batch after which profiling should end.

   ``sync_timings``
      Specifies whether Determined should wait for all GPU kernel streams before considering a
      timing as ended. Defaults to 'true'. Applies only for frameworks that collect timing metrics
      (currently just PyTorch).

.. _experiment-configuration_training_units:

****************
 Training Units
****************

Some configuration settings, such as searcher training lengths and budgets,
``min_validation_period``, and ``min_checkpoint_period``, can be expressed in terms of a few
training units: records, batches, or epochs.

-  ``records``: A *record* is a single labeled example (sometimes called a sample).

-  ``batches``: A *batch* is a group of records. The number of records in a batch is configured via
   the ``global_batch_size`` hyperparameter.

-  ``epoch``: An *epoch* is a single copy of the entire training data set; the number of records in
   an epoch is configured via the :ref:`records_per_epoch <config-records-per-epoch>` configuration
   field.

For example, to specify the ``max_length`` for a searcher in terms of batches, the configuration
would read as shown below.

.. code:: yaml

   max_length:
     batches: 900

To express it in terms of records or epochs, ``records`` or ``epochs`` would be specified in place
of ``batches``. In the case of epochs, :ref:`records_per_epoch <config-records-per-epoch>` must also
be specified. Below is an example that configures a ``single`` searcher to train a model for 64
epochs.

.. code:: yaml

   records_per_epoch: 50000
   searcher:
     name: single
     metric: validation_error
     max_length:
       epochs: 64
     smaller_is_better: true

The configured :ref:`records_per_epoch <config-records-per-epoch>` is only used for interpreting
configuration fields that are expressed in epochs. Actual epoch boundaries are still determined by
the dataset itself (specifically, the end of an epoch occurs when the training data loader runs out
of records).

.. note::

   If the amount of data to train a model on is specified using records or epochs and the batch size
   does not divide evenly into the configured number of inputs, the remaining "partial batch" of
   data will be dropped (ignored). For example, if an experiment is configured to train a single
   model on 10 records with a configured batch size of 3, the model will only be trained on 9
   records of data. In the corner case that a trial is configured to be trained for less than a
   single batch of data, a single complete batch will be used instead.

Caveats
=======

In most cases, a value expressed using one type of training unit can be converted to a different
type of training unit with identical behavior, with a few caveats:

-  Because training units must be positive integers, converting between quantities of different
   types is not always possible. For example, converting 50 ``records`` into batches is not possible
   if the batch size is 64.

-  When doing a hyperparameter search over a range of values for ``global_batch_size``, the
   specified ``batches`` cannot be converted to a fixed number of records or epochs and hence cause
   different behaviors in different trials of the search.

-  When using :ref:`adaptive_asha <experiment-configuration-searcher-adaptive>`, a single training
   unit is treated as atomic (unable to be divided into fractional parts) when dividing
   ``max_length`` into the series of rounds (or rungs) by which we early-stop underperforming
   trials. This rounding may result in unexpected behavior when configuring ``max_length`` in terms
   of a small number of large epochs or batches.

To verify your search is working as intended before committing to a full run, you can use the CLI's
"preview search" feature:

.. code::

   det preview-search <configuration.yaml>

.. _slurm-config:

***************
 Slurm Options
***************

The ``slurm`` section specifies configuration options applicable when the cluster is configured with
:ref:`resource_manager.type: slurm <cluster-configuration-slurm>`.

**Optional Fields**

``gpu_type``
   An optional GPU type name to be included in the generated Slurm ``--gpus`` or ``--gres`` option
   if you have configured GPU types within your Slurm gres configuration. Specify this option to
   select that specific GPU type when there are multiple GPU types within the Slurm partition. The
   default is to select GPUs without regard to their type. For example, you can request the
   ``tesla`` GPU type with:

   .. code:: yaml

      slurm:
         gpu_type: tesla

``sbatch_args``
   Additional Slurm options to be passed when launching trials with ``sbatch``. These options enable
   control of Slurm options not otherwise managed by Determined. For example, to specify required
   memory per cpu and exclusive access to an entire node when scheduled, you could specify:

   .. code:: yaml

      slurm:
         sbatch_args:
            - --mem-per-cpu=10
            - --exclusive

``slots_per_node``
   The minimum number of slots required for a node to be scheduled during a trial. If
   :ref:`gres_supported <cluster-configuration-slurm>` is false, specify ``slots_per_node`` in order
   to utilize more than one GPU per node. It is the user’s responsibility to ensure that
   ``slots_per_node`` GPUs will be available on nodes selected for the job using other
   configurations such as targeting a specific resource pool with only GPU nodes or specifying a
   Slurm constraint in the experiment configuration.

.. _pbs-config:

*************
 PBS Options
*************

The ``pbs`` section specifies configuration options applicable when the cluster is configured with
:ref:`resource_manager.type: pbs <cluster-configuration-slurm>`.

**Optional Fields**

``pbsbatch_args``
   Additional PBS options to be passed when launching trials with ``qsub``. These options enable
   control of PBS options not otherwise managed by Determined. For example, to specify that the job
   should have a priority of ``1000`` and a project name of ``MyProjectName``, you could specify:

   .. code:: yaml

      pbs:
         pbsbatch_args:
            - -p1000
            - -PMyProjectName

   Requesting of resources and job placement may be influenced through use of ``-l``, however chunk
   count, chunk arrangement, and GPU or CPU counts per chunk (depending on the value of
   ``slot_type``) are controlled by Determined; any values specified for these quantities will be
   ignored. Consider if the following were specified for a CUDA experiment:

   .. code:: yaml

      pbs:
         pbsbatch_args:
            - -l select=2:ngpus=4:mem=4gb
            - -l place=scatter:shared
            - -l walltime=1:00:00

   The chunk count (two), the GPU count per chunk (four), and the chunk arrangement (scatter) will
   all be ignored in favor of values calculated by Determined.

``slots_per_node``
   The minimum number of slots required for a node to be scheduled during a trial. If
   :ref:`gres_supported <cluster-configuration-slurm>` is false, specify ``slots_per_node`` in order
   to utilize more than one GPU per node. It is the user’s responsibility to ensure that
   ``slots_per_node`` GPUs will be available on the nodes selected for the job using other
   configurations such as targeting a specific resource pool with only ``slots_per_node`` GPU nodes
   or specifying a PBS constraint in the experiment configuration.
