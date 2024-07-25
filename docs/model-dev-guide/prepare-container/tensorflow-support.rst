.. _tensorflow-support:

####################
 TensorFlow Support
####################

************************
 TensorFlow Core Models
************************

Determined supports for TensorFlow models using the :ref:`Keras <api-keras-ug>` API. For models that
use low-level TensorFlow Core APIs, we recommend wrapping your model in Keras as suggested by the
official `TensorFlow <https://www.tensorflow.org/guide/basics#training_loops>`_ documentation.

*******************
 TensorFlow 1 vs 2
*******************

Determined supports both TensorFlow 1 and 2. The version of TensorFlow used for a particular
experiment is controlled by the configured container image. Determined provides prebuilt Docker
images that include TensorFlow 2+, 1.15, and 2.8, respectively:

-  ``determinedai/tensorflow-ngc:0.35.0``
-  ``determinedai/environments:cuda-10.2-pytorch-1.7-tf-1.15-gpu-0.21.2``
-  ``determinedai/environments:cuda-11.2-tf-2.8-gpu-0.29.1``

Lightweight CPU-only counterparts are also available:

-  ``determinedai/environments:py-3.8-tf-2.8-cpu-0.29.1``

To change the container image used for an experiment, specify :ref:`environment.image
<exp-environment-image>` in the experiment configuration file. Please see :ref:`container-images`
for more details about configuring training environments and a more complete list of prebuilt Docker
images.
