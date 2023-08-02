.. _core-getting-started:

#####################
 Core API User Guide
#####################

.. meta::
   :description: Learn how to use the flexible Core API to train any deep learning model. This user guide walks you through plugging in your existing training code to get up and running.

This guide will help you get up and running with the Core API.

+------------------------------------------------------------------+
| Visit the API reference                                          |
+==================================================================+
| :ref:`core-reference`                                            |
+------------------------------------------------------------------+

In this user guide, we'll show you how to adapt model training code to use the Core API. As an
example, we'll be working with the PyTorch MNIST model.

************
 Objectives
************

These step-by-step instructions walk you through modifying a script for the purpose of performing
the following functions:

-  Report metrics
-  Report checkpoints
-  Perform a hyperparameter search
-  Perform distributed training

After completing the steps in this user guide, you will be able to:

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

Access the tutorial files via the :download:`core_api_pytorch_mnist.tgz
</examples/core_api_pytorch_mnist.tgz>` download link or directly from the `Github repository
<https://github.com/determined-ai/determined/tree/master/examples/tutorials/core_api_pytorch_mnist>`_.
These scripts have already been modified to fit the steps outlined in this tutorial.

In this initial step, we’ll run our experiment using the ``model_def.py`` script and its
accompanying ``const.yaml`` experiment configuration file.

CD into the directory and run this command:

.. code:: bash

   det e create const.yaml . -f

.. note::

   ``det e create const.yaml . -f`` instructs Determined to follow the logs of the first trial that
   is created as part of the experiment. The command will stay active and display the live output
   from the logs of the first trial as it progresses.

Open the Determined WebUI by navigating to the master URL. One way to do this is to navigate to
``http://localhost:8080/``, accept the default Determined username, leave the password empty, and
click **Sign In**.

.. note::

   This tutorial provides instructions for running a local distributed training job. Your setup may
   be different. For example, for instructions on how to run a remote distributed training job,
   visit the :ref:`qs-mdldev`.

In the WebUI, select your experiment. You'll notice the tabs do not yet contain any information. In
the next section, we'll report training and validation metrics.

************************
 Step 2: Report Metrics
************************

To report training and validation metrics to the Determined master, we’ll add a few lines of code to
our script. More specifically, we'll create a :class:`~determined.core.Context` object to allow
interaction with the master. Then, we'll pass the ``core_context`` as an argument into ``main()``,
``train()``, and ``test()`` and modify the function headers accordingly.

To run our experiment, we'll use the ``model_def_metrics.py`` script and its accompanying
``metrics.yaml`` experiment configuration file.

.. include:: ../../_shared/note-premade-tutorial-script.txt

Begin by importing Determined:

.. code:: python

   import determined as det

Step 2.1: Modify the Main Loop
==============================

We'll need a :class:`~determined.core.Context` object for interacting with the master. To accomplish
this, we'll modify the ``__main__`` loop to include ``core_context``:

.. note::

   Refer to the ``if __name__ == "__main__":`` block in ``model_def_metrics.py``

.. literalinclude:: ../../../examples/tutorials/core_api_pytorch_mnist/model_def_metrics.py
   :language: python
   :start-after: # Docs snippet start: modify main loop core context
   :end-before: # Docs snippet end: modify main loop core content
   :dedent:

Step 2.2: Modify the Train Method
=================================

Use ``core_context.train`` to report training and validation metrics.

#. Begin by importing the ``determined`` module:

.. literalinclude:: ../../../examples/tutorials/core_api_pytorch_mnist/model_def_metrics.py
   :language: python
   :start-after: # Docs snippet start: report training metrics
   :end-before: # Docs snippet end: report training metrics
   :dedent:

and ``core_context.train.report_validation_metrics()``:

.. literalinclude:: ../../../examples/tutorials/core_api_pytorch_mnist/model_def_metrics.py
   :language: python
   :start-after: # Docs snippet start: report validation metrics
   :end-before: # Docs snippet end: report validation metrics
   :dedent:

Step 2.3: Modify the Test Method
================================

Modify the ``test()`` function header to include ``args`` and other elements you’ll need during the
evaluation loop. The ``args`` variable lets you pass configuration settings such as batch size and
learning rate. In addition, pass the newly created ``core_context`` into both ``train()`` and
``test()``. Passing ``core_context`` enables reporting of metrics to the Determined master.

.. literalinclude:: ../../../examples/tutorials/core_api_pytorch_mnist/model_def_metrics.py
   :language: python
   :start-after: # Docs snippet start: pass core context
   :end-before: # Docs snippet end: pass core context
   :dedent:

Create a ``steps_completed`` variable to plot metrics on a graph in the WebUI:

.. literalinclude:: ../../../examples/tutorials/core_api_pytorch_mnist/model_def_metrics.py
   :language: python
   :start-after: # Docs snippet start: calculate steps completed
   :end-before: # Docs snippet end: calculate steps completed
   :dedent:

Step 2.4: Run the Experiment
============================

Run the following command to run the experiment:

.. code::

   det e create metrics.yaml .

Open the Determined WebUI again and navigate to the **Overview** tab.

The WebUI now displays metrics. In this step, you learned how to add a few new lines of code in
order to report training and validation metrics to the Determined master. Next, we’ll modify our
script to report checkpoints.

***********************
 Step 3: Checkpointing
***********************

Checkpointing periodically during training and reporting the checkpoints to the master gives us the
ability to stop and restart training. In this section, we’ll modify our script for the purpose of
checkpointing.

In this step, we’ll run our experiment using the ``model_def_checkpoints.py`` script and its
accompanying ``checkpoints.yaml`` experiment configuration file.

.. include:: ../../_shared/note-premade-tutorial-script.txt

Step 3.1: Save Checkpoints
==========================

To save checkpoints, add the ``store_path`` function to your script:

.. literalinclude:: ../../../examples/tutorials/core_api_pytorch_mnist/model_def_checkpoints.py
   :language: python
   :start-after: # Docs snippet start: save checkpoint
   :end-before: # Docs snippet end: save checkpoint
   :dedent:

Step 3.2: Continuations
=======================

There are two types of continuations: pausing and reactivating training using the WebUI or clicking
**Continue Trial** after the experiment completes.

These two types of continuations have different behaviors. While you always want to preserve the
model's state, you do not always want to preserve the batch index. When you pause and reactivate you
want training to continue from the same batch index, but when starting a fresh experiment you want
training to start with a fresh batch index. You can save the trial ID in the checkpoint and use it
to distinguish the two types of continuations.

To distinguish between the two types of continuations, you can save the trial ID in the checkpoint.

**Enable Pausing and Resuming an Experiment**

To enable pausing an experiment, enable preemption:

.. literalinclude:: ../../../examples/tutorials/core_api_pytorch_mnist/model_def_checkpoints.py
   :language: python
   :start-after: # Docs snippet start: enable preemption
   :end-before: # Docs snippet end: enable preemption
   :dedent:

Define a load_state function for restarting model training from existing checkpoint:

.. literalinclude:: ../../../examples/tutorials/core_api_pytorch_mnist/model_def_checkpoints.py
   :language: python
   :start-after: # Docs snippet start: define load state to restart
   :end-before: # Docs snippet end: define load state to restart
   :dedent:

If checkpoint exists, load it and assign it to model state prior to resuming training:

.. literalinclude:: ../../../examples/tutorials/core_api_pytorch_mnist/model_def_checkpoints.py
   :language: python
   :start-after: # Docs snippet start: if checkpoint assign to model state
   :end-before: # Docs snippet end: if checkpoint assign to model state
   :dedent:

**Enable Continuing the Trial**

To enable continuing the trial after the experiment completes, save the trial ID. One way to do this
is to load the checkpoint and save the checkpoint in a file in the checkpoint directory.

Open the ``checkpoint.pt`` file in binary mode and compare ``ckpt_trial_id`` with the current
``trial_id``:

.. literalinclude:: ../../../examples/tutorials/core_api_pytorch_mnist/model_def_checkpoints.py
   :language: python
   :start-after: # Docs snippet start: compare checkpoint and current trial IDs
   :end-before: # Docs snippet end: compare checkpoint and current trial IDs
   :dedent:

Save the checkpoint in the ``checkpoint.pt`` file:

.. literalinclude:: ../../../examples/tutorials/core_api_pytorch_mnist/model_def_checkpoints.py
   :language: python
   :start-after: # Docs snippet start: save checkpoint
   :end-before: # Docs snippet end: save checkpoint
   :dedent:

Detect when the experiment is paused by the WebUI:

.. literalinclude:: ../../../examples/tutorials/core_api_pytorch_mnist/model_def_checkpoints.py
   :language: python
   :start-after: # Docs snippet start: enable preemption
   :end-before: # Docs snippet end: enable preemption
   :dedent:

Step 3.3: Run the Experiment
============================

Run the following command to run the experiment:

.. code:: bash

   det e create checkpoints.yaml . -f

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
   </model-dev-guide/hyperparameter/search-methods/overview>`.

-  ``metric``: ``test_loss``

-  ``smaller_is_better``: ``True`` (This is equivalent to minimization vs. maximization of
   objective.)

-  ``max_trials``: 500 (This is the maximum number of trials the searcher should run.)

-  ``max_length``: 20 epochs (The max length of a trial. For more information, visit Adaptive ASHA
   in the :doc:`Experiment Configuration Reference
   </reference/training/experiment-config-reference>`.

In addition, we also need to define the hyperparameters themselves. Adaptive ASHA will pick values
between the ``minval`` and ``maxval`` for each hyperparameter for each trial.

.. note::

   To see early stopping in action, try setting ``max_trials`` to over 500 and playing around with
   the hyperparameter search values.

In this step, we’ll run our experiment using the ``model_def_adaptive.py`` script and its
accompanying ``adaptive.yaml`` experiment configuration file.

.. include:: ../../_shared/note-premade-tutorial-script.txt

Begin by accessing the hyperparameters in your code:

.. literalinclude:: ../../../examples/tutorials/core_api_pytorch_mnist/model_def_adaptive.py
   :language: python
   :start-after: # Docs snippet start: get hparams
   :end-before: # Docs snippet end: get hparams
   :dedent:

Then, pass the hyperparameters into your model and optimizer:

.. literalinclude:: ../../../examples/tutorials/core_api_pytorch_mnist/model_def_adaptive.py
   :language: python
   :start-after: # Docs snippet start: pass hyperparameters
   :end-before: # Docs snippet end: pass hyperparameters
   :dedent:

Ensure your model is set to use the selected values on a per-trial basis rather than your previously
hardcoded values:

.. literalinclude:: ../../../examples/tutorials/core_api_pytorch_mnist/model_def_adaptive.py
   :language: python
   :start-after: # Docs snippet start: per trial basis
   :end-before: # Docs snippet end: per trial basis
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

In this example, we’ll be using PyTorch’s DistributedDataParallel. We’ll also need to make specific
changes to our configuration experiment file.

In this step, we’ll run our experiment using the ``model_def_distributed.py`` script and its
accompanying ``distributed.yaml`` experiment configuration file.

.. include:: ../../_shared/note-premade-tutorial-script.txt

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
   :start-after: # Docs snippet start: import torch distrib
   :end-before: # Docs snippet end: import torch distrib
   :dedent:

Initialize a process group using ``torch``. After initializing a process group, initialize a
Determined distributed context using ``from_torch_distributed``:

.. literalinclude:: ../../../examples/tutorials/core_api_pytorch_mnist/model_def_distributed.py
   :language: python
   :start-after: # Docs snippet start: initialize process group
   :end-before: # Docs snippet end: initialize process group
   :dedent:

In ``main``, set your selected device to the device with index of ``local_rank``. This is a best
practice even if you only have a single GPU-per-node setup:

.. note::

   Refer to the ``if use_cuda:`` block in ``model_def_distributed.py``

.. literalinclude:: ../../../examples/tutorials/core_api_pytorch_mnist/model_def_distributed.py
   :language: python
   :start-after: # Docs snippet start: set device
   :end-before: # Docs snippet end: set device
   :dedent:

Shard the data into ``num_replicas`` non-overlapping parts. ``num_replicas`` is equal to
``core_context.distributed.size``, or the number of slots:

.. literalinclude:: ../../../examples/tutorials/core_api_pytorch_mnist/model_def_distributed.py
   :language: python
   :start-after: # Docs snippet start: shard data
   :end-before: # Docs snippet end: shard data
   :dedent:

Wrap your model with torch’s DistributedDataParallel:

.. literalinclude:: ../../../examples/tutorials/core_api_pytorch_mnist/model_def_distributed.py
   :language: python
   :start-after: # Docs snippet start: DDP
   :end-before: # Docs snippet end: DDP
   :dedent:

Finally, at each place in the code where you upload checkpoints, report training metrics, or report
progress to the master, make sure this is done only on rank 0, e.g.,:

.. literalinclude:: ../../../examples/tutorials/core_api_pytorch_mnist/model_def_distributed.py
   :language: python
   :start-after: # Docs snippet start: report metrics rank 0
   :end-before: # Docs snippet end: report metrics rank 0
   :dedent:

Step 5.3: Run the Experiment
============================

Run the following command to run the experiment:

.. code:: bash

   det e create distributed.yaml .

In the Determined WebUI, navigate to the **Cluster** pane.

You should be able to see multiple slots active corresponding to the value you set for
``slots_per_trial`` you set in ``distributed.yaml``, as well as logs appearing from multiple ranks.

************
 Next Steps
************

In this user guide, you learned how to use the Core API to integrate a model into Determined. You
also saw how to modify a training script and use the appropriate configuration file to report
metrics and checkpointing, perform a hyperparameter search, and run distributed training.

.. include:: ../../_shared/note-dtrain-learn-more.txt
