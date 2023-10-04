.. _api-deepspeed-ug:

###############
 DeepSpeed API
###############

.. meta::
   :description: Learn how you can train your model in Determined using the DeepSpeed engine.

In this guide, you'll learn how to use the DeepSpeed API.

+-----------------------------------------------------------------------+
| Visit the API reference                                               |
+=======================================================================+
| :doc:`/reference/training/api-deepspeed-reference`                    |
+-----------------------------------------------------------------------+

`DeepSpeed <https://deepspeed.ai/>`_ is a Microsoft library that supports large-scale, distributed
learning with sharded optimizer state training and pipeline parallelism. Determined supports
DeepSpeed with the :class:`~determined.pytorch.deepspeed.DeepSpeedTrial` API.
:class:`~determined.pytorch.deepspeed.DeepSpeedTrial` provides a way to use an automated training
loop with DeepSpeed.

Determined DeepSpeed documentation:

-  :ref:`Usage Guide <deepspeed-api>` guides you through how to subclass
   :class:`~determined.pytorch.deepspeed.DeepSpeedTrial` for your own training experiments.

-  :ref:`Advanced Usage <deepspeed-advanced>` discusses advanced topics like using multiple model
   engines, manual gradient aggregation, custom data loaders, and custom model parallelism.

-  :ref:`PyTorchTrial to DeepSpeedTrial <pytorch-to-deepspeed>` covers how to convert an existing
   :class:`~determined.pytorch.PyTorchTrial` to
   :class:`~determined.pytorch.deepspeed.DeepSpeedTrial`.

-  :ref:`DeepSpeed Autotune: User Guide <deepspeed-autotuning>` demonstrates how to use DeepSpeed
   Autotune to take full advantage of your hardware and model.

-  :ref:`API Reference <deepspeed-reference>` lays out the classes and methods related to DeepSpeed
   support including the full API specification for
   :class:`~determined.pytorch.deepspeed.DeepSpeedTrial` and
   :class:`~determined.pytorch.deepspeed.DeepSpeedTrialContext`.

.. toctree::
   :maxdepth: 1
   :hidden:
   :glob: 

   ./*
