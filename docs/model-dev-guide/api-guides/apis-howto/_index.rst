.. _apis-howto-overview:

###############
 Training APIs
###############

.. meta::
   :description: You can train almost any deep learning model using the Determined Training APIs. By using the Training API guides, you'll discover how to take your existing model code and train your model in Determined. Each API guide contains a link to its corresponding API reference.

You can train almost any deep learning model using the Determined Training APIs. The Training API
guides describe how to take your existing model code and train your model in Determined. Each API
guide contains a link to its corresponding API reference.

**********
 Core API
**********

The Core API is a low-level, flexible API that lets you train models in any deep learning framework.
With the Core API, you can plug in your existing training code. You'll then use an :ref:`experiment
configuration <experiment-configuration>` to tell Determined how to train the model - e.g.,
multi-GPU, hyperparameter search, etc.

-  :ref:`api-core-ug-basic`: Increment an Integer
-  :ref:`core-getting-started`: Train a Model

.. _high-level-apis:

*****************
 High-Level APIs
*****************

The Trial APIs offer higher-level integrations with popular deep learning frameworks. With the Trial
APIs, you first convert your existing training code by subclassing a Trial class and implementing
methods that define each component of training - e.g., model architecture, data loader, optimizer,
learning rate scheduler, callbacks, etc. This is called the Trial definition. With the code
structured in this way, Determined is able to run the training loop and provide advanced training
and model management capabilities.

Once you have converted your code, you can use an :ref:`experiment configuration
<experiment-configuration>` to tell Determined how to train the model - e.g., multi-GPU,
hyperparameter search, etc.

-  :ref:`api-pytorch-ug`
-  :ref:`api-keras-ug`
-  :ref:`api-deepspeed-ug`

Looking for a Basic Tutorial?
=============================

If you'd like to review how to implement the Determined APIs on simple models, visit our
:ref:`tutorials-index`.

Prefer to use an Example Model?
===============================

If you'd like to build off of an existing model that already runs on Determined, visit our
:ref:`example-solutions` to see if the model you'd like to train is already available.

AMD ROCm Support
================

For AMD ROCm support, visit :ref:`rocm-support`

.. toctree::
   :caption: Training APIs
   :hidden:
   :glob:

   ./*
   ./*/_index
