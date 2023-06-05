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
-  How configuration templates can help reduce redundancy

.. toctree::
   :caption: Distributed Training
   :hidden:

   Distributed Training Concepts <dtrain-introduction>
   Implementing Distributed Training <dtrain-implement>
