.. _use-trained-models:

#############
 Checkpoints
#############

.. meta::
   :description: Gain an understanding about working with model checkpoints including querying checkpoints from trials and experiments.
   :keywords: checkpoints, Python, checkpoint APIs

Determined provides APIs for downloading checkpoints and loading them into memory in a Python
process.

This guide discusses:

#. Querying model checkpoints from trials and experiments.
#. Loading model checkpoints in Python.
#. Storing additional user-defined metadata in a checkpoint.
#. Using the Determined CLI to download checkpoints to disk.

The Checkpoint Export API is a subset of the features found in the
:mod:`~determined.experimental.client` module.

**********************
 Querying Checkpoints
**********************

The :class:`~determined.experimental.client.Experiment` class is a reference to an experiment.
Within this class, there is a :meth:`~determined.experimental.client.Experiment.list_checkpoints`
method. When called without arguments, this method looks at the experiment configuration's searcher
field for the `metric` and `smaller_is_better` values. These values are used to sort the
experiment's checkpoints by validation performance. As an example, the searcher settings in the
following experiment configuration file snippet will result in checkpoints being sorted by the loss
metric in ascending order.

.. code:: yaml

   searcher:
     metric: "loss"
     smaller_is_better: true

The following snippet of Python code can be run after the specified experiment has generated a
checkpoint. It gets a list of sorted :class:`~determined.experimental.client.Checkpoint` instances
associated with the experiment and selects the one with the best validation metric.

.. code:: python

   from determined.experimental import client

   checkpoint = client.list_checkpoints(id)[0]

Checkpoints can be sorted by any metric using the ``sort_by`` keyword argument, which defines which
metric to use, and ``order_by``, which defines whether to sort the checkpoints in ascending or
descending order with respect to the specified metric.

.. code:: python

   from determined.experimental import checkpoint, client

   checkpoint = (
       client.get_experiment(id).list_checkpoints(sort_by="accuracy", order_by=checkpoint.CheckpointOrderBy.DESC)
   )

You may also query checkpoints by preset checkpoint parameters, as defined by
:class:`~determined.experimental.checkpoint.CheckpointSortBy`. The following code snippet fetches
all checkpoints for the experiment sorted by the checkpoint's trial ID in descending order.

.. code:: python

   from determined.experimental import checkpoint, client

   checkpoints = client.get_experiment(id).list_checkpoints(
      sort_by=checkpoint.CheckpointSortBy.TRIAL_ID,
      order_by=checkpoint.CheckpointOrderBy.DESC
   )

:class:`~determined.experimental.client.Trial` is used for fine-grained control over checkpoint
selection within a trial. It contains a
:meth:`~determined.experimental.client.Trial.list_checkpoints` method, which mirrors
:meth:`~determined.experimental.client.Experiment.list_checkpoints` for an experiment.

The following snippet showcases how to select the latest checkpoint for a trial.

.. code:: python

   from determined.experimental import checkpoint, client

   trial = client.get_trial(id)

   latest_checkpoint = trial.list_checkpoints(
      sort_by=checkpoint.CheckpointSortBy.END_TIME,
      order_by=checkpoint.CheckpointOrderBy.DESC,
      limit=1
   )[0]

   specific_checkpoint = client.get_checkpoint(uuid="uuid-for-checkpoint")

********************************
 Using the ``Checkpoint`` Class
********************************

The :class:`~determined.experimental.client.Checkpoint` class can both download the checkpoint from
persistent storage and load it into memory in a Python process.

The :meth:`~determined.experimental.client.Checkpoint.download` method downloads a checkpoint from
persistent storage to a directory on the local file system. By default, checkpoints are downloaded
to ``checkpoints/<checkpoint-uuid>/`` (relative to the current working directory). The
:meth:`~determined.experimental.client.Checkpoint.download` method accepts ``path`` as an optional
parameter, which changes the checkpoint download location.

.. code:: python

   from determined.experimental import client

   checkpoint = client.get_experiment(id).top_checkpoint()
   checkpoint_path = checkpoint.download()

   specific_path = checkpoint.download(path="specific-checkpoint-path")

The :meth:`~determined.experimental.client.Checkpoint.load` method downloads the checkpoint, if it
does not already exist locally, and loads it into memory. The return type and behavior is different
depending on whether you are using TensorFlow or PyTorch.

PyTorch Checkpoints
===================

When using PyTorch models, the :meth:`~determined.experimental.client.Checkpoint.load` method
returns a parameterized instance of your trial class as defined in the experiment config under the
:ref:`entrypoint <experiment-config-entrypoint>` field. The trained model can then be accessed from
the ``model`` attribute of the ``Trial`` object, as shown in the following snippet.

.. code:: python

   from determined.experimental import client
   from determined import pytorch

   checkpoint = client.get_experiment(id).top_checkpoint()
   path = checkpoint.download()
   trial = pytorch.load_trial_from_checkpoint_path(path)
   model = trial.model

   predictions = model(samples)

PyTorch checkpoints are saved using `pickle <https://docs.python.org/3/library/pickle.html>`__ and
loaded as :ref:`api-pytorch-ug` objects (see `the PyTorch documentation
<https://pytorch.org/docs/stable/notes/serialization.html>`__ for details).

TensorFlow Checkpoints
======================

When using TensorFlow models, the :meth:`~determined.experimental.client.Checkpoint.load` method
returns a compiled model with weights loaded. This will be the same TensorFlow model returned by
your ``build_model()`` method defined in your trial class specified by the experiment config
:ref:`entrypoint <experiment-config-entrypoint>` field. The trained model can then be used to make
predictions as shown in the following snippet.

.. code:: python

   from determined.experimental import client
   from determined import keras

   checkpoint = client.get_experiment(id).top_checkpoint()
   path = checkpoint.download()
   model = keras.load_model_from_checkpoint_path(path)

   predictions = model(samples)

TensorFlow checkpoints are saved in either the ``saved_model`` or ``h5`` formats and are loaded as
trackable objects (see documentation for `tf.compat.v1.saved_model.load_v2
<https://www.tensorflow.org/versions/r1.15/api_docs/python/tf/saved_model/load_v2>`__ for details).

.. _store-checkpoint-metadata:

*****************************************
 Adding User-Defined Checkpoint Metadata
*****************************************

You can add arbitrary user-defined metadata to a checkpoint via the Python SDK. This feature is
useful for storing post-training metrics, labels, information related to deployment, etc.

.. code:: python

   from determined.experimental import client

   checkpoint = client.get_experiment(id).top_checkpoint()
   checkpoint.add_metadata({"environment": "production"})

   # Metadata will be stored in Determined and accessible on the checkpoint object.
   print(checkpoint.metadata)

You may store an arbitrarily nested dictionary using the
:meth:`~determined.experimental.client.Checkpoint.add_metadata` method. If the top level key already
exists the entire tree beneath it will be overwritten.

.. code:: python

   from determined.experimental import client

   checkpoint = client.get_experiment(id).top_checkpoint()
   checkpoint.add_metadata({"metrics": {"loss": 0.12}})
   checkpoint.add_metadata({"metrics": {"acc": 0.92}})

   print(checkpoint.metadata)  # Output: {"metrics": {"acc": 0.92}}

You may remove metadata via the :meth:`~determined.experimental.client.Checkpoint.remove_metadata`
method. The method accepts a list of top level keys. The entire tree beneath the keys passed will be
deleted.

.. code:: python

   from determined.experimental import client

   checkpoint = client.get_experiment(id).top_checkpoint()
   checkpoint.remove_metadata(["metrics"])

***************************************
 Downloading Checkpoints using the CLI
***************************************

:ref:`The Determined CLI <cli-ug>` can be used to view all the checkpoints associated with an
experiment:

.. code:: bash

   $ det experiment list-checkpoints <experiment-id>

Checkpoints are saved to external storage, according to the :ref:`checkpoint_storage
<checkpoint-storage>` section in the experiment configuration. Each checkpoint has a UUID, which is
used as the name of the checkpoint directory on the external storage system. For example, if the
experiment is configured to save checkpoints to a shared file system:

.. code:: yaml

   checkpoint_storage:
     type: shared_fs
     host_path: /mnt/nfs-volume-1

A checkpoint with UUID ``b3ed462c-a6c9-41e9-9202-5cb8ff00e109`` can be found in the directory
``/mnt/nfs-volume-1/b3ed462c-a6c9-41e9-9202-5cb8ff00e109``.

Determined offers the following CLI commands for downloading checkpoints locally:

#. ``det checkpoint download``
#. ``det trial download``
#. ``det experiment download``

.. warning::

   When downloading checkpoints in a shared file system, we assume the same shared file system is
   mounted locally.

The ``det checkpoint download`` command downloads a checkpoint for the given UUID as shown below:

.. code::

   # Download a specific checkpoint.
   det checkpoint download 46985143-af68-4d48-ab91-a6447052ca49

The command should display output resembling the following upon successfully downloading the
checkpoint.

.. code::

   Local checkpoint path:
   checkpoints/46985143-af68-4d48-ab91-a6447052ca49

        Batch | Checkpoint UUID                      | Validation Metrics
   -----------+--------------------------------------+---------------------------------------------
         1000 | 46985143-af68-4d48-ab91-a6447052ca49 | {
              |                                      |     "num_inputs": 0,
              |                                      |     "validation_metrics": {
              |                                      |         "loss": 7.906739711761475,
              |                                      |         "accuracy": 0.9646000266075134,
              |                                      |         "global_step": 1000,
              |                                      |         "average_loss": 0.12492649257183075
              |                                      |     }
              |                                      | }

The ``det trial download`` command downloads checkpoints for a specified trial. Similar to the
:class:`~determined.experimental.client.Trial` API, the ``det trial download`` command accepts
``--best``, ``--latest``, and ``--uuid`` options.

.. code::

   # Download best checkpoint.
   det trial download <trial_id> --best
   # Download best checkpoint to a particular directory.
   det trial download <trial_id> --best --output-dir local_checkpoint

The command should display output resembling the following upon successfully downloading the
checkpoint.

.. code::

   Local checkpoint path:
   checkpoints/46985143-af68-4d48-ab91-a6447052ca49

        Batch | Checkpoint UUID                      | Validation Metrics
   -----------+--------------------------------------+---------------------------------------------
         1000 | 46985143-af68-4d48-ab91-a6447052ca49 | {
              |                                      |     "num_inputs": 0,
              |                                      |     "validation_metrics": {
              |                                      |         "loss": 7.906739711761475,
              |                                      |         "accuracy": 0.9646000266075134,
              |                                      |         "global_step": 1000,
              |                                      |         "average_loss": 0.12492649257183075
              |                                      |     }
              |                                      | }

The ``--latest`` and ``--uuid`` options are used as follows:

.. code:: bash

   # Download the most recent checkpoint.
   det trial download <trial_id> --latest

   # Download a specific checkpoint.
   det trial download <trial_id> --uuid <uuid-for-checkpoint>

Finally, the ``det experiment download`` command provides a similar experience to using the
:class:`Python SDK <determined.experimental.client.Experiment>`.

.. code:: bash

   # Download the best checkpoint for a given experiment.
   det experiment download <experiment_id>

   # Download the best 3 checkpoints for a given experiment.
   det experiment download <experiment_id> --top-n 3

The command should display output resembling the following upon successfully downloading the
checkpoints.

.. code::

   Local checkpoint path:
   checkpoints/8d45f621-8652-4268-8445-6ae9a735e453

        Batch | Checkpoint UUID                      | Validation Metrics
   -----------+--------------------------------------+------------------------------------------
          400 | 8d45f621-8652-4268-8445-6ae9a735e453 | {
              |                                      |     "num_inputs": 56,
              |                                      |     "validation_metrics": {
              |                                      |         "val_loss": 0.26509127765893936,
              |                                      |         "val_categorical_accuracy": 1
              |                                      |     }
              |                                      | }

   Local checkpoint path:
   checkpoints/62131ba1-983c-49a8-98ef-36207611d71f

        Batch | Checkpoint UUID                      | Validation Metrics
   -----------+--------------------------------------+------------------------------------------
         1600 | 62131ba1-983c-49a8-98ef-36207611d71f | {
              |                                      |     "num_inputs": 50,
              |                                      |     "validation_metrics": {
              |                                      |         "val_loss": 0.04411194706335664,
              |                                      |         "val_categorical_accuracy": 1
              |                                      |     }
              |                                      | }

   Local checkpoint path:
   checkpoints/a36d2a61-a384-44f7-a84b-8b30b09cb618

        Batch | Checkpoint UUID                      | Validation Metrics
   -----------+--------------------------------------+------------------------------------------
          400 | a36d2a61-a384-44f7-a84b-8b30b09cb618 | {
              |                                      |     "num_inputs": 46,
              |                                      |     "validation_metrics": {
              |                                      |         "val_loss": 0.07265569269657135,
              |                                      |         "val_categorical_accuracy": 1
              |                                      |     }
              |                                      | }

************
 Next Steps
************

-  :ref:`python-sdk-reference`: The reference documentation for this API.
-  :ref:`organizing-models`
