.. _multi-gpu-training:

######################################
 Distributed Training with Determined
######################################

Learn how to perform optimized distributed training with Determined to speed up the training of a
single trial.

In :ref:`Concepts of Distributed Training <multi-gpu-training-concept>`, you'll learn about the
following topics:

-  How Determined distributed training works
-  Reducing computation and communication overhead
-  Training effectively with large batch sizes
-  Model characteristics that affect performance
-  Debugging performance bottlenecks
-  Optimizing training

Visit :ref:`Implementing Distributed Training <multi-gpu-training-implement>` to discover how to
implement distributed training, including the following:

-  Connectivity considerations for multi-machine training
-  Configuration including slots per trial and global batch size
-  Considerations for concurrent data downloads
-  Details to be aware regarding scheduler behavior
-  Accelerating inference workloads

Additional Resources:

-  Learn how :ref:`configuration templates <config-template>` can help reduce redundancy.
-  Discover how Determined aims to support reproducible machine learning experiments in
   :ref:`Reproducibility <reproducibility>`.
-  In :ref:`Optimizing Training <optimizing-training>`, you'll learn about out-of-the box tools you
   can use for instrumenting training.

.. toctree::
   :caption: Distributed Training
   :hidden:

   Distributed Training Concepts <dtrain-introduction>
   Implementing Distributed Training <dtrain-implement>
   Configuration Templates <config-templates>
   Reproducibility <reproducibility>
   Optimizing Training <optimize-training>
