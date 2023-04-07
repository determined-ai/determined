#####################
 Core API User Guide
#####################

.. meta::
   :description: Learn how to train almost any machine learning model using Determined AI's Core API. This guide covers metrics tracking, checkpoint tracking and preemption support, hyperparameter search, and distributing work across multiple GPUs or nodes.

In this guide, you'll learn how to use the Core API.

+------------------------------------------------------------------+
| Visit the API reference                                          |
+==================================================================+
| :ref:`core-reference`                                            |
+------------------------------------------------------------------+

   :description: Learn how to train almost any machine learning model using Determined AI's Core API. This guide covers metrics tracking, checkpoint tracking and preemption support, hyperparameter search, and distributing work across multiple GPUs or nodes.

In this user guide, we'll show you how to adapt model training code to use the Core API. As an
With the Core API you can train arbitrary models on the Determined platform with seamless access to
the following capabilities:

************
 Objectives
************

These step-by-step instructions walk you through modifying a script for the purpose of performing
the following functions:

-  Report metrics
-  Report checkpoints
-  Perform a hyperparameter search
-  Perform distributed training

After completing the steps in this user guide, you will be able to do the following:

-  Understand the minimum requirements for running an experiment
-  Modify a script and an experiment configuration file
-  Understand how to convert model code
-  Use the Core API to train a model

***************
 Prerequisites
***************

**Required**

-  A Determined cluster

**Recommended**

-  :ref:`qs-mdldev`

*****************************************************
 Step 1: Get the Tutorial Files & Run the Experiment
*****************************************************

To run an experiment, you need, at minimum, a script and an experiment configuration (YAML) file.

Create a new directory.

Get the files from the :download:`core_api_pytorch_mnist.tgz </examples/core_api_pytorch_mnist.tgz>`
download or from the `Github repository
<https://github.com/determined-ai/determined/tree/master/examples/tutorials/core_api_pytorch_mnist>`_.

.. note::

   Throughout this user guide, we’ll show you how to make modifications to your script. After each
   step, we’ll re-run our experiment using the appropriate script and its accompanying experiment
   configuration file.

In this initial step, we’ll run our experiment using the ``model_def.py`` script and its
accompanying ``const.yaml`` experiment configuration file.

CD into the directory and run this command:

.. code:: bash

   det e create const.yaml . -f

Open the Determined WebUI by navigating to ``http://localhost:8080/``. Accept the default determined
username, leave the password empty, and click **Sign In**.

In the WebUI, select your experiment, and then navigate to the **Logs** tab.

************************
 Step 2: Report Metrics
************************

To report training and validation metrics to the Determined master, we’ll add a few lines of code to
our script. More specifically, we'll create a :class:`~determined.core.Context` object to allow
interaction with the master. Then, we'll pass the ``core_context`` as an argument into ``main()``,
``train()``, and ``test()`` and modify the function headers accordingly.

To run our experiment, we'll use the ``model_def_metrics.py`` script and its accompanying
``metrics.yaml`` experiment configuration file.

Begin by importing Determined:

.. code:: python

   import determined as det

Step 2.1: Modify the Main Loop
==============================

We'll need a ``core.Context`` object for interacting with the master. To accomplish this, we'll
modify the __main__loop to include ``core_context``:

.. note::

   Refer to the ``if __name__ == "__main__":`` block in ``model_def_metrics.py``

.. literalinclude:: ../../../examples/tutorials/core_api_pytorch_mnist/model_def_metrics.py
   :language: python
   :start-after: # Docs snippet start after: modify main loop core context
   :end-at: main(core_context=core_context)

Step 2.2: Modify the Train Method
=================================

Use ``core_context.train`` to report training and validation metrics.

Modify the train() method by adding ``core_context.train.report_training_metrics()``:

.. literalinclude:: ../../../examples/tutorials/core_api_pytorch_mnist/model_def_metrics.py
   :language: python
   :start-after: # Docs snippet start after: report training metrics
   :end-before: # Docs snippet end before: report training metrics
   :dedent:

and ``core_context.train.report_validation_metrics()``:

.. literalinclude:: ../../../examples/tutorials/core_api_pytorch_mnist/model_def_metrics.py
   :language: python
   :start-after: # Docs snippet start after: report validation metrics
   :end-before: # Docs snippet end before: report validation metrics
   :dedent:

.. note::

The complete ``1_metrics.py`` and ``1_metrics.yaml`` listings used in this example can be found in
the :download:`core_api.tgz </examples/core_api.tgz>` download or in the `GitHub repository
<https://github.com/determined-ai/determined/tree/master/examples/tutorials/core_api>`_.

   You can modify your code to report epoch-based metrics as shown in this example. When you report
   epoch-based metrics during training, you'll be able to view epoch-based metric data in the WebUI.

.. tip::

   Avoiding a Duplicate Key Error

   It is best to stitch metrics together in one continuous graph as your experiment progresses. To
   accomplish this, calculate the index used for ``steps_completed`` appropriately. The appropriate
   calculation avoids re-writing metrics with the same index each time a new epoch begins and avoids
   a duplicate key error.

After an epoch value has been reported, **Epoch** will be an available option for the X-Axis when
viewing the metric data graph in the WebUI.

Step 2.3: Modify the Test Method
================================

Modify the ``test()`` function header to include ``args`` and other elements you’ll need during the
evaluation loop. In addition, pass the newly created ``core_context`` into both ``train()`` and
``test()``:

.. literalinclude:: ../../../examples/tutorials/core_api_pytorch_mnist/model_def_metrics.py
   :language: python
   :start-after: # Docs snippet start after: pass core context
   :end-before: # Docs snippet end before: pass core context
   :dedent:

Create a ``steps_completed`` variable to plot metrics on a graph in the WebUI:

.. literalinclude:: ../../../examples/tutorials/core_api_pytorch_mnist/model_def_metrics.py
   :language: python
   :start-after: # Docs snippet start after: calculate steps completed
   :end-before: # Docs snippet end before: calculate steps completed
   :dedent:

Step 2.4: Run the Experiment
============================

Run the following command to run the experiment:

.. code::

   det e create metrics.yaml .

Open the Determined WebUI again and navigate to the **Overview** tab.

#. Create a ``2_checkpoints.yaml`` file by copying the ``0_start.yaml`` file and changing the first
   couple of lines:

   .. literalinclude:: ../../../examples/tutorials/core_api/2_checkpoints.yaml
      :language: yaml
      :lines: 1-2

#. Run the code using the command:

   .. code:: bash

      det e create 2_checkpoints.yaml . -f

#. You can navigate to the experiment in the WebUI and pause it mid-training. The trial shuts down
   and stops producing logs. If you reactivate training it resumes where it stopped. After training
   is completed, click Continue Trial to see that fresh training is started but that the model
   weight continues from where previous training finished.

The complete ``2_checkpoints.py`` and ``2_checkpoints.yaml`` listings used in this example can be
found in the :download:`core_api.tgz </examples/core_api.tgz>` download or in the `GitHub repository
<https://github.com/determined-ai/determined/tree/master/examples/tutorials/core_api>`_.

.. _core-hpsearch:

***********************
 Step 3: Checkpointing
***********************

Checkpointing periodically during training and reporting the checkpoints to the master gives us the
ability to stop and restart training. In this section, we’ll modify our script for the purpose of
checkpointing.

In this step, we’ll run our experiment using the ``model_def_checkpoints.py`` script and its
accompanying ``checkpoints.yaml`` experiment configuration file.

Step 3.1: Save Checkpoints
==========================

To save checkpoints, add the ``store_path`` function to your script:

.. literalinclude:: ../../../examples/tutorials/core_api_pytorch_mnist/model_def_checkpoints.py
   :language: python
   :start-after: # Docs snippet start after: save checkpoint
   :end-before: # Docs snippet end before: save checkpoint
   :dedent:

Step 3.2: Continuations
=======================

There are two types of continuations: pausing and reactivating training using the WebUI or clicking
**Continue Trial** after the experiment completes.

Each type of continuation has its own behavior. While you always want to preserve the value you are
incrementing (the “model weight”), you do not always want to preserve the batch index. When you
pause and reactivate you want training to continue from the same batch index. However, when starting
a fresh experiment, you want training to start with a fresh batch index.

To distinguish between the two types of continuations, you can save the trial ID in the checkpoint.

**Enable Pausing and Resuming an Experiment**

To enable pausing an experiment, enable preemption:

.. literalinclude:: ../../../examples/tutorials/core_api_pytorch_mnist/model_def_checkpoints.py
   :language: python
   :start-after: # Docs snippet start after: enable preemption
   :end-before: # Docs snippet end before: enable preemption
   :dedent:

Define a load_state function for restarting model training from existing checkpoint:

.. literalinclude:: ../../../examples/tutorials/core_api_pytorch_mnist/model_def_checkpoints.py
   :language: python
   :start-after: # Docs snippet start after: define load state to restart
   :end-before: # Docs snippet end before: define load state to restart
   :dedent:

If checkpoint exists, load it and assign it to model state prior to resuming training:

.. literalinclude:: ../../../examples/tutorials/core_api_pytorch_mnist/model_def_checkpoints.py
   :language: python
   :start-after: # Docs snippet start after: if checkpoint assign to model state
   :end-before: # Docs snippet end before: if checkpoint assign to model state
   :dedent:

**Enable Continuing the Trial**

To enable continuing the trial after the experiment completes, save the trial ID. One way to do this
is to load the checkpoint and save the checkpoint in a file in the checkpoint directory.

Open the `checkpoint.pt` file in binary mode and compare `ckpt_trial_id` with the current
`trial_id`:

.. literalinclude:: ../../../examples/tutorials/core_api_pytorch_mnist/model_def_checkpoints.py
   :language: python
   :start-after: # Docs snippet start after: compare checkpoint and current trial IDs
   :end-before: # Docs snippet end before: compare checkpoint and current trial IDs
   :dedent:

Save the checkpoint in the `checkpoint.pt` file:

.. literalinclude:: ../../../examples/tutorials/core_api_pytorch_mnist/model_def_checkpoints.py
   :language: python
   :start-after: # Docs snippet start after: save checkpoint
   :end-before: # Docs snippet end before: save checkpoint
   :dedent:

Detect when the experiment is paused by the WebUI:

.. literalinclude:: ../../../examples/tutorials/core_api_pytorch_mnist/model_def_checkpoints.py
   :language: python
   :start-after: # Docs snippet start after: enable preemption
   :end-before: # Docs snippet end before: enable preemption
   :dedent:

Step 3.3: Run the Experiment
============================

Run the following command to run the experiment:

.. code:: bash

   det e create checkpoints.yaml . -f

The ``-f`` option is the short form of ``--follow``.

In the Determined WebUI, nagivate to the **Checkpoints** tab.

Checkpoints are saved and deleted according to the default
:ref:`experiment-config-checkpoint-policy`. You can modify the checkpoint policy in the experiment
configuration file.

*******************************
 Step 4: Hyperparameter Search
*******************************

With the Core API you can run advanced hyperparameter searches with arbitrary training code. The
hyperparameter search logic is in the master, which coordinates many different Trials. Each trial
runs a train-validate-report loop:

.. table::

   +----------+--------------------------------------------------------------------------+
   | Train    | Train until a point chosen by the hyperparameter search algorithm and    |
   |          | obtained via the Core API.  The length of training is absolute, so you   |
   |          | have to keep track of how much you have already trained to know how much |
   |          | more to train.                                                           |
   +----------+--------------------------------------------------------------------------+
   | Validate | Validate your model to obtain the metric you configured in the           |
   |          | ``searcher.metric`` field of your experiment config.                     |
   +----------+--------------------------------------------------------------------------+
   | Report   | Use the Core API to report results to the master.                        |
   +----------+--------------------------------------------------------------------------+

To perform a hyperparameter search, we'll update our script to define the hyperparameter search
settings we want to use for our experiment. More specifically, we'll need to define the following
settings in our experiment configuration file:

-  ``name:`` ``adaptive_asha`` (name of our searcher. For all options, visit :doc:`Search Methods
   </training/hyperparameter/search-methods/overview>`.

-  ``metric``: ``test_loss``

-  ``smaller_is_better``: ``True`` (This is equivalent to minimization vs. maximization of
   objective.)

-  ``max_trials``: 500 (This is the maximum number of trials the searcher should run.)

-  ``max_length``: 20 epochs (The max length of a trial. For more information, visit Adaptive ASHA
   in the :doc:`Experiment Configuration Reference
   </reference/reference-training/experiment-config-reference>`.

In addition, we also need to define the hyperparameters themselves. Adaptive ASHA will pick values
between the ``minval`` and ``maxval`` for each hyperparameter for each trial.

.. note::

   To see early stopping in action, try setting ``max_trials`` to over 500 and playing around with
   the hyperparameter search values.

In this step, we’ll run our experiment using the ``model_def_adaptive.py`` script and its
accompanying ``adaptive.yaml`` experiment configuration file.

Begin by accessing the hyperparameters in your code:

.. literalinclude:: ../../../examples/tutorials/core_api_pytorch_mnist/model_def_adaptive.py
   :language: python
   :start-after: # Docs snippet start after: get hparams
   :end-before: # Docs snippet end before: get hparams
   :dedent:

Then, pass the hyperparameters into your model and optimizer:

.. literalinclude:: ../../../examples/tutorials/core_api_pytorch_mnist/model_def_adaptive.py
   :language: python
   :start-after: # Docs snippet start after: pass hyperparameters
   :end-before: # Docs snippet end before: pass hyperparameters
   :dedent:

The complete ``3_hpsearch.py`` and ``3_hpsearch.yaml`` listings used in this example can be found in
the :download:`core_api.tgz </examples/core_api.tgz>` download or in the `GitHub repository
<https://github.com/determined-ai/determined/tree/master/examples/tutorials/core_api>`_.

.. literalinclude:: ../../../examples/tutorials/core_api_pytorch_mnist/model_def_adaptive.py
   :language: python
   :start-after: # Docs snippet start after: per trial basis
   :end-before: # Docs snippet end before: per trial basis
   :dedent:

Step 4.1: Run the Experiment
============================

Run the following command to run the experiment:

.. code:: bash

   det e create adaptive.yaml .

In the Determined WebUI, navigate to the **Hyperparameters** tab.

You should see a graph in the WebUI that displays the various trials initiated by the Adaptive ASHA
hyperparameter search algorithm.

******************************
 Step 5: Distributed Training
******************************

The Core API has special features for running distributed training. Some of the more important
features are:

-  Access to all IP addresses of every node in the Trial (through the ClusterInfo API).

-  Communication primitives such as :meth:`~determined.core.DistributedContext.allgather`,
   :meth:`~determined.core.DistributedContext.gather`, and
   :meth:`~determined.core.DistributedContext.broadcast` to give you out-of-the-box coordination
   between workers.

-  Since many distributed training frameworks expect all workers in training to operate in-step, the
   :meth:`~determined.core.PreemptContext.should_preempt` call is automatically synchronized across
   workers so that all workers decide to preempt or continue as a unit.

.. tip::

   Launchers

   Typically, you do not have to write your own launcher. Determined provides launchers for Horovod,
   torch.distributed, and DeepSpeed. For more information about launcher options, visit
   :ref:`experiments`.

To perform distributed training with the Core API, you’ll need to use the appropriate distributed
training library *before* creating a Determined :class:`~determined.core.DistributedContext`.

In this example, we’ll be using PyTorch’s DistributedDataParallel. We’ll also need to make specific
changes to our configuration experiment file.

In this step, we’ll run our experiment using the ``model_def_distributed.py`` script and its
accompanying ``distributed.yaml`` experiment configuration file.

Step 5.1: Edit Your Experiment Configuration File
=================================================

Edit your experiment configuration file to point to a launch script:

.. code:: yaml

   entrypoint: >-
      python3 -m determined.launch.torch_distributed
      python3 model_def_distributed.py

and, set ``slots_per_trial`` (under ``resources``) to the number of GPUs you want to distribute the
training across:

.. code:: yaml

   resources:
     slots_per_trial: 4

Step 5.2: Modify Your Training Script
=====================================

Add a few more imports to your training script:

.. literalinclude:: ../../../examples/tutorials/core_api_pytorch_mnist/model_def_distributed.py
   :language: python
   :start-after: # Docs snippet start after: import torch distrib
   :end-before: # Docs snippet end before: import torch distrib
   :dedent:

Initialize a process group and a Determined distributed context using ``from_torch_distributed``:

.. literalinclude:: ../../../examples/tutorials/core_api_pytorch_mnist/model_def_distributed.py
   :language: python
   :start-after: # Docs snippet start after: initialize process group
   :end-before: # Docs snippet end before: initialize process group
   :dedent:

In ``main``, set your selected device to the device with index of ``local_rank``. This is a best
practice even if you only have a single GPU-per-node setup:

.. note::

   Refer to the ``if use_cuda:`` block in ``model_def_distributed.py``

.. literalinclude:: ../../../examples/tutorials/core_api_pytorch_mnist/model_def_distributed.py
   :language: python
   :start-after: # Docs snippet start after: set device
   :end-before: # Docs snippet end before: set device
   :dedent:

Shard the data into ``num_replicas`` non-overlapping parts. ``num_replicas`` is equal to
``core_context.distributed.size``, or the number of slots:

.. literalinclude:: ../../../examples/tutorials/core_api_pytorch_mnist/model_def_distributed.py
   :language: python
   :start-after: # Docs snippet start after: shard data
   :end-before: # Docs snippet end before: shard data
   :dedent:

Wrap your model with torch’s DistributedDataParallel:

.. literalinclude:: ../../../examples/tutorials/core_api_pytorch_mnist/model_def_distributed.py
   :language: python
   :start-after: # Docs snippet start after: DDP
   :end-before: # Docs snippet end before: DDP
   :dedent:

The complete ``4_distributed.py`` and ``3_hpsearch.yaml`` listings used in this example can be found
in the :download:`core_api.tgz </examples/core_api.tgz>` download or in the `GitHub repository
<https://github.com/determined-ai/determined/tree/master/examples/tutorials/core_api>`_.
