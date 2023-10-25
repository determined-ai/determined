.. _transition-managed-determined:

########################################
 Transition a Script from Detached Mode
########################################

.. meta::
   :description: Discover how to transition from detached mode to leveraging Determined's features such as distributed training.

Now that you know how to run a workload via :ref:`detached mode <detached-mode-index>` and submit
metrics associated with it to Determined, you are ready to transition to submitting the workload to
Determined. Submitting the workload to Determined will give you access to features such as
distributed training, hyperparameter search, and resource management.

************
 Objectives
************

These step-by-step instructions will cover:

-  Preparing an experiment configuration file to transition a script from detached mode
-  Submitting the experiment to Determined

By the end of this guide, you'll:

-  Understand the process to transition from detached mode
-  Grasp the benefits of leveraging Determined's resource management and distributed training
   capabilities

***************
 Prerequisites
***************

**Required**

-  A Determined cluster
-  Familiarity with configuration files
-  :ref:`distributed-training-checkpointing`

**Recommended**

-  :ref:`simple-metrics-reporting`
-  :ref:`save-load-checkpoints`

***************************************************
 Step 1: Prepare the Experiment Configuration File
***************************************************

To run an experiment, you need, at minimum, a script and an experiment configuration (YAML) file.

We'll use the script we created in :ref:`distributed-training-checkpointing` and create a new
experiment configuration file.

Use the following code to create the experiment configuration file:

.. code:: yaml

   name: unmanaged-3-torch-distributed
   # Here we use Determined's PyTorch distributed launcher.
   # You don't need to specify details about the cluster topology; Determined
   # takes care of that for you.
   # Simply specify resource requirements like resources.slots_per_trial.
   entrypoint: >-
     python3 -m determined.launch.torch_distributed
     python3 3_torch_distributed.py

   resources:
     slots_per_trial: 2

   # Use the single searcher to run just one instance of the training script.
   searcher:
      name: single
       # metric is required but it shouldn't hurt to ignore it at this point.
      metric: x
      # max_length is ignored if the training script ignores it.
      max_length: 1

   max_restarts: 0

*********************************************
 Step 2: Submit the Experiment to Determined
*********************************************

Navigate to the directory containing your model code and use the following command to run the
experiment:

.. code:: bash

   det e create -m <MASTER_IP:PORT> exp_config_yaml .

.. note::

   To run the experiment on a machine with a single CPU or GPU, set ``slots_per_trial`` to ``1``.

This command will submit your experiment to Determined, leveraging its managed mode functionalities.

Navigate to ``<DET_MASTER_IP:PORT>`` in your web browser to see the experiment.

************
 Next Steps
************

Congratulations! You've successfully transitioned a script from detached mode to being managed by
Determined, tapping into Determined's resource management and distributed training capabilities. To
learn more about Determined, visit the :ref:`tutorials <tutorials-index>`.
