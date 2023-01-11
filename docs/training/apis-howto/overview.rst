###############
 Training APIs
###############

This section describes how to use the training APIs and contains the API reference information.

Determined leverages specific APIs for each Deep Learning framework. In general, users convert their
existing training code by subclassing a Trial class and implementing methods that advertise
components of the user's model - e.g., model architecture, data loader, optimizer, learning rate
scheduler, callbacks, etc. This is called the Trial definition and by structuring their code in this
way, Determined is able to run the training loop and provide advanced training and model management
capabilities.

Once users' model code are ported to Determined's APIs, they can use an :doc:`experiment
configuration </reference/reference-training/experiment-config-reference>` to configure how
Determined should train the model - e.g., multi-GPU, hyperparameter search, etc.

If you have existing model code that you'd like to train with Determined, continue to one of the API
docs below depending on your ML Framework.

-  :doc:`/training/apis-howto/api-core/overview`
-  :doc:`/training/apis-howto/api-pytorch-ug`
-  :doc:`/training/apis-howto/api-pytorch-lightning-ug`
-  :doc:`/training/apis-howto/api-keras-ug`
-  :doc:`/training/apis-howto/deepspeed/overview`

If you'd like a review of implementing the Determined APIs on simple models, please take a look at
our :doc:`Tutorials </tutorials/pytorch-mnist-tutorial>`. Or, if you'd like to build off of an
existing model that already runs on Determined, take a look at our :doc:`examples
</example-solutions/examples>` to see if the model you'd like to train is already available.

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

-  ``determinedai/environments:cuda-11.3-pytorch-1.10-tf-2.8-gpu-0.19.10`` (default)
-  ``determinedai/environments:cuda-10.2-pytorch-1.7-tf-1.15-gpu-0.19.10``
-  ``determinedai/environments:cuda-11.2-tf-2.7-gpu-0.19.10``

We also provide lightweight CPU-only counterparts:

-  ``determinedai/environments:py-3.8-pytorch-1.10-tf-2.8-cpu-0.19.10``
-  ``determinedai/environments:py-3.7-pytorch-1.7-tf-1.15-cpu-0.19.10``
-  ``determinedai/environments:py-3.8-tf-2.7-cpu-0.19.10``

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

-  ``determinedai/environments:rocm-5.0-pytorch-1.10-tf-2.7-rocm-0.19.10``

Known limitations:

-  Only agent-based deployments are available; Kubernetes is not yet supported.
-  GPU profiling is not yet supported.

.. toctree::
   :caption: Training
   :hidden:

   api-core/overview
   api-pytorch-ug
   api-pytorch-lightning-ug
   api-keras-ug
   deepspeed/overview
   api-estimator-ug
