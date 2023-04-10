###############
 Training APIs
###############

You can train almost any deep learning model using the Determined Training APIs. The Training API
guides describe how to take your existing model code and train your model in Determined. Each API
guide contains a link to its corresponding API reference.

**********
 Core API
**********

The Core API is a low-level, flexible API that lets you train models in any deep learning framework.
With the Core API, you can plug in your existing training code. You'll then use an :doc:`experiment
configuration </reference/reference-training/experiment-config-reference>` to tell Determined how to
train the model - e.g., multi-GPU, hyperparameter search, etc.

-  :doc:`/training/apis-howto/api-core-ug`

*****************
 High-Level APIs
*****************

The Trial APIs offer higher-level integrations with popular deep learning frameworks. With the Trial
APIs, you first convert your existing training code by subclassing a Trial class and implementing
methods that define each component of training - e.g., model architecture, data loader, optimizer,
learning rate scheduler, callbacks, etc. This is called the Trial definition. With the code
structured in this way, Determined is able to run the training loop and provide advanced training
and model management capabilities.

Once you have converted your code, you can use an :doc:`experiment configuration
</reference/reference-training/experiment-config-reference>` to tell Determined how to train the
model - e.g., multi-GPU, hyperparameter search, etc.

-  :doc:`/training/apis-howto/api-pytorch-ug`
-  :doc:`/training/apis-howto/api-pytorch-lightning-ug`
-  :doc:`/training/apis-howto/api-keras-ug`
-  :doc:`/training/apis-howto/deepspeed/overview`

Looking for a Basic Tutorial?
=============================

If you'd like to review how to implement the Determined APIs on simple models, visit our `Tutorials
<https://docs.determined.ai/latest/tutorials/index.html>`_.

Prefer to use an Example Model?
===============================

If you'd like to build off of an existing model that already runs on Determined, visit our
:doc:`examples </example-solutions/examples>` to see if the model you'd like to train is already
available.

********************
 TensorFlow Support
********************

TensorFlow Core Models
======================

Determined has support for TensorFlow models that use the :doc:`Keras
</training/apis-howto/api-keras-ug>` or :doc:`Estimator </training/apis-howto/api-estimator-ug>`
APIs. For models that use the low-level TensorFlow Core APIs, we recommend wrapping your model in
Keras, as recommended by the official `TensorFlow
<https://www.tensorflow.org/guide/basics#training_loops>`_ documentation.

TensorFlow 1 vs 2
=================

Determined supports both TensorFlow 1 and 2. The version of TensorFlow that is used for a particular
experiment is controlled by the container image that has been configured for that experiment.
Determined provides prebuilt Docker images that include TensorFlow 2.8, 1.15, and 2.7, respectively:

-  ``determinedai/environments-dev:cuda-11.3-pytorch-1.12-tf-2.8-gpu-0.21.2`` (default)
-  ``determinedai/environments-dev:cuda-10.2-pytorch-1.7-tf-1.15-gpu-0.21.2``
-  ``determinedai/environments-dev:cuda-11.2-tf-2.7-gpu-0.21.2``

We also provide lightweight CPU-only counterparts:

-  ``determinedai/environments-dev:py-3.8-pytorch-1.12-tf-2.8-cpu-0.21.2``
-  ``determinedai/environments-dev:py-3.7-pytorch-1.7-tf-1.15-cpu-0.21.2``
-  ``determinedai/environments-dev:py-3.8-tf-2.7-cpu-0.21.2``

To change the container image used for an experiment, specify :ref:`environment.image
<exp-environment-image>` in the experiment configuration file. Please see :ref:`container-images`
for more details about configuring training environments and a more complete list of prebuilt Docker
images.

******************
 AMD ROCm Support
******************

.. _rocm-support:

Determined has experimental support for ROCm. Determined provides a prebuilt Docker image that
includes ROCm 4.2, PyTorch 1.9 and Tensorflow 2.5:

-  ``determinedai/environments-dev:rocm-5.0-pytorch-1.10-tf-2.7-rocm-0.21.2``

Known limitations:

-  Only agent-based deployments are available; Kubernetes is not yet supported.
-  GPU profiling is not yet supported.

.. toctree::
   :caption: Training
   :hidden:

   api-core-ug
   api-pytorch-ug
   api-pytorch-lightning-ug
   api-keras-ug
   deepspeed/overview
   api-estimator-ug
