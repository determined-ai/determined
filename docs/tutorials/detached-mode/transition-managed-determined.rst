.. _transition-managed-determined:

##################################
 Transition to Managed Determined
##################################

.. meta::
   :description: Discover how to transition from standalone training to leveraging Determined's managed mode. This guide highlights the process of moving to Determined's distributed training and resource management.

Now that you know how to do something with workloads via :ref:`detached mode <detached-mode-index>`,
you are ready to transition to submitting your workloads to Determined.

This will give you access to features such as distributed training, hyperparameter search, and
resource management.

************
 Objectives
************

These step-by-step instructions will cover:

-  Preparing a configuration YAML file for Determined's managed mode
-  Submitting the experiment to Determined

By the end of this guide, you'll:

-  Understand the process to transition to Determined's managed mode
-  Grasp the benefits of leveraging Determined's resource management and distributed training
   capabilities

***************
 Prerequisites
***************

**Required**

-  A Determined cluster
-  Familiarity with YAML configurations

**Recommended**

-  `Simple Metrics Reporting User Guide <simple-metrics-reporting>`_
-  `Save and Load State using Checkpoints User Guide <save-load-checkpoints>`_
-  `Use Distributed Training with Sharded Checkpointing User Guide
   <distributed-training-checkpointing>`_

***************************************************************
 Step 1: Prepare the Configuration YAML for Managed Determined
***************************************************************

Use the following structure for the YAML file:

.. code:: yaml

   name: unmanaged-3-torch-distributed
   entrypoint: >-
     python3 -m determined.launch.torch_distributed
     python3 3_torch_distributed.py

   resources:
     slots_per_trial: 2

   searcher:
      name: single
      metric: x
      max_length: 1

   max_restarts: 0

Note: With Determined's torch distributed launcher, you don't need to specify details about the
cluster topology. Determined manages this for you. Simply specify resource requirements in the YAML.

*********************************************
 Step 2: Submit the Experiment to Determined
*********************************************

Navigate to the directory containing your model code and use the following command:

.. code:: bash

   det e create -m <MASTER_IP:PORT> exp_config_yaml .

This command will submit your experiment to Determined, leveraging its managed mode functionalities.

************
 Next Steps
************

Congratulations! You've successfully transitioned to Determined's managed mode, tapping into its
robust resource management and distributed training capabilities. With these skills in hand, you're
ready to conduct larger and more complex training sessions efficiently.
