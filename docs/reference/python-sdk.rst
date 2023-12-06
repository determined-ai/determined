.. _python-sdk:

############
 Python SDK
############

You can interact with a Determined cluster using the Python SDK. The Determined Python SDK, a part
of the broader Determined Python library, is designed to perform tasks such as:

-  Creating and organizing experiments
-  Downloading model checkpoints, and adding them to the model registry
-  Retrieving trial metrics

The client module exposes many of the same capabilities as the det CLI tool directly to Python code
with an object-oriented interface.

*****************************************
 Example: Run and Administer Experiments
*****************************************

This `demo <https://github.com/determined-ai/determined-examples/tree/main/blog/python_sdk_demo>`__
describes how to run a script that shows an example of running and administering experiments. This
example uses MedMNIST datasets and demos the following:

-  Archiving existing experiments with the same names as the datasets being train on.
-  Creating models for each dataset and registering them in the Determined model registry.
-  Training a model for each dataset by creating an experiment.
-  Registering the best checkpoint for each experiment in the Determined model registry.

For instructions on running this example, visit `det-python-sdk-demo
<https://github.com/determined-ai/determined-examples/tree/main/blog/python_sdk_demo>`__ in the
`Determined GitHub repo <https://github.com/determined-ai/determined/tree/master/examples>`__.

*********************************************
 Example: Find the Top Performing Checkpoint
*********************************************

In this example, we'll walk through the most basic workflow for creating an experiment, waiting for
it to complete, and finding the top-performing checkpoint.

The first step is to import the client module and possibly to call login():

.. code:: python

   from determined.experimental import client

   # We will assume that you have called `det user login`, so this is unnecessary:
   # client.login(master=..., user=..., password=...)

The next step is to call create_experiment():

.. code:: python

   # Config can be a path to a config file or a Python dict of the config.
   exp = client.create_experiment(config="my_config.yaml", model_dir=".")
   print(f"started experiment {exp.id}")

The returned object is an ``Experiment`` object, which offers methods to manage the experiment's
lifecycle. In the following example, we simply await the experiment's completion.

.. code:: python

   exit_status = exp.wait()
   print(f"experiment completed with status {exit_status}")

Now that the experiment has completed, you can grab the top-performing checkpoint from training:

.. code:: python

   best_checkpoint = exp.list_checkpoints()[0]
   print(f"best checkpoint was {best_checkpoint.uuid}")

.. _python-sdk-reference:

Python SDK Reference
====================

************
 ``Client``
************

.. automodule:: determined.experimental.client
   :members: login, create_experiment, get_experiment, get_trial, get_checkpoint, create_model, get_model, get_models, iter_trials_metrics
   :member-order: bysource

*************
 ``OrderBy``
*************

.. autoclass:: determined.experimental.client.OrderBy
   :members:
   :member-order: bysource

****************
 ``Checkpoint``
****************

.. autoclass:: determined.experimental.client.Checkpoint
   :members:
   :member-order: bysource

****************
 ``Determined``
****************

.. autoclass:: determined.experimental.client.Determined
   :members:
   :exclude-members: stream_trials_metrics, stream_trials_training_metrics, stream_trials_validation_metrics
   :member-order: bysource

****************
 ``Experiment``
****************

.. autoclass:: determined.experimental.client.Experiment
   :members:
   :exclude-members: get_trials
   :member-order: bysource

******************
 ``DownloadMode``
******************

.. autoclass:: determined.experimental.client.DownloadMode
   :members:
   :member-order: bysource

***********
 ``Model``
***********

.. autoclass:: determined.experimental.client.Model
   :members:
   :exclude-members: get_metrics
   :member-order: bysource

*****************
 ``ModelSortBy``
*****************

.. autoclass:: determined.experimental.client.ModelSortBy
   :members:
   :member-order: bysource

******************
 ``ModelVersion``
******************

.. autoclass:: determined.experimental.model.ModelVersion
   :members:
   :member-order: bysource

*************
 ``Project``
*************

.. autoclass:: determined.experimental.client.Project
   :members:
   :member-order: bysource

***********
 ``Trial``
***********

.. autoclass:: determined.experimental.client.Trial
   :members:
   :exclude-members: stream_metrics, stream_training_metrics, stream_validation_metrics
   :member-order: bysource

******************
 ``TrialMetrics``
******************

.. autoclass:: determined.experimental.client.TrialMetrics

***************
 ``Workspace``
***************

.. autoclass:: determined.experimental.client.Workspace
   :members:
   :member-order: bysource
