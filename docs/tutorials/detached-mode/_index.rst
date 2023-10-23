.. _detached-mode-index:

##########################
 How to Use Detached Mode
##########################

Detached mode is an enhancement to the Core API that allows you to:

-  Log metrics in Determined while running your workload *anywhere*.
-  Maintain full control of your workload-associated metrics and metadata.
-  Report metrics from multiple jobs rather than a single experiment.
-  Track experiments using Determined notebooks.

This means you can try out Determined's experiment tracking and visualization features and then
transition to submitting your workloads to Determined when you want to perform distributed training,
hyperparameter search, and resource management.

***********
 Tutorials
***********

To get started with detached mode, try the following tutorials:

.. note::

   If you have not yet installed Determined, visit the :ref:`quick installation guide <basic>` to
   get up and running quickly.

-  :ref:`simple-metrics-reporting`
-  :ref:`save-load-checkpoints`
-  :ref:`distributed-training-checkpointing`
-  :ref:`transition-managed-determined`

.. toctree::
   :hidden:

   Simple Metrics Reporting <simple-metrics-reporting>
   Save and Load States Using Checkpoints <save-load-checkpoints>
   Distributed Training With Sharded Checkpointing <distributed-training-checkpointing>
   Transition From Detached Mode <transition-managed-determined>
