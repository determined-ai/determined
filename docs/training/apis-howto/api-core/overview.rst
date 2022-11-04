##########
 Core API
##########

+------------------------------------------------------------------+
| API reference                                                    |
+==================================================================+
| :doc:`/reference/reference-training/training/api-core-reference` |
+------------------------------------------------------------------+

With the Core API you can train arbitrary models on the Determined platform with seamless access to
the the following capabilities:

-  metrics tracking
-  checkpoint tracking and preemption support
-  hyperparameter search
-  distributing work across multiple GPUs and/or nodes

These are the same features provided by the higher-level PyTorchTrial, DeepSpeedTrial, and
TFKerasTrial APIs: those APIs are implemented using the Core API.

This section shows you how to get started using the Core API.

.. toctree::
   :maxdepth: 1
   :hidden:

   getting-started
   metrics
   checkpoints
   hpsearch
   distributed
