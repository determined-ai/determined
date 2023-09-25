.. _python-sdk:

############
 Python SDK
############

You can interact with a Determined cluster with the Python SDK.

The client module exposes many of the same capabilities as the det CLI tool directly to Python code
with an object-oriented interface.

*****************************
 Experiment Workflow Example
*****************************

As a simple example, let’s walk through the most basic workflow for creating an experiment, waiting
for it to complete, and finding the top-performing checkpoint.

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

The returned object will be an ``ExperimentReference`` object, which has methods for controlling the
lifetime of the experiment running on the cluster. In this example, we will just wait for the
experiment to complete.

.. code:: python

   exit_status = exp.wait()
   print(f"experiment completed with status {exit_status}")

Now that the experiment has completed, you can grab the top-performing checkpoint from training:

.. code:: python

   best_checkpoint = exp.top_checkpoint()
   print(f"best checkpoint was {best_checkpoint.uuid}")

.. _python-sdk-reference:

**********************
 Python SDK Reference
**********************

``Client``
==========

.. automodule:: determined.experimental.client
   :members: login, create_experiment, get_experiment, get_trial, get_checkpoint, create_model, get_model, get_models, stream_trials_metrics, stream_trials_training_metrics, stream_trials_validation_metrics
   :member-order: bysource

``Checkpoint``
==============

.. autoclass:: determined.experimental.client.Checkpoint
   :members:
   :member-order: bysource

``Determined``
==============

.. autoclass:: determined.experimental.client.Determined
   :members:
   :member-order: bysource

``ExperimentReference``
=======================

.. autoclass:: determined.experimental.client.ExperimentReference
   :members:
   :member-order: bysource

``DownloadMode``
================

.. autoclass:: determined.experimental.client.DownloadMode
   :members:
   :member-order: bysource

``Model``
=========

.. autoclass:: determined.experimental.client.Model
   :members:
   :member-order: bysource

``ModelOrderBy``
================

.. autoclass:: determined.experimental.client.ModelOrderBy
   :members:
   :member-order: bysource

``ModelSortBy``
===============

.. autoclass:: determined.experimental.client.ModelSortBy
   :members:
   :member-order: bysource

``ModelVersion``
================

.. autoclass:: determined.experimental.model.ModelVersion
   :members:
   :member-order: bysource

``TrialReference``
==================

.. autoclass:: determined.experimental.client.TrialReference
   :members:
   :member-order: bysource

``TrainingMetrics``
===================

.. autoclass:: determined.experimental.client.TrainingMetrics
   :members:
   :member-order: bysource

``ValidationMetrics``
=====================

.. autoclass:: determined.experimental.client.ValidationMetrics
   :members:
   :member-order: bysource
