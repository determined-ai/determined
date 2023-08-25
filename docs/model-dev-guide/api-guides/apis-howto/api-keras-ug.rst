.. _api-keras-ug:

###########
 Keras API
###########

.. meta::
   :description: Learn how to use the Keras API to train a Keras model. This user guide walks you through loading your data, defining the model, customizing how the model.fit function is called, checkpointing, and callbacks.

In this guide, you'll learn how to use the Keras API.

+---------------------------------------------------------------------+
| Visit the API reference                                             |
+=====================================================================+
| :doc:`/reference/training/api-keras-reference`                      |
+---------------------------------------------------------------------+

This document guides you through training a Keras model in Determined. You need to implement a trial
class that inherits :class:`~determined.keras.TFKerasTrial` and specify it as the entrypoint in the
:doc:`experiment configuration </reference/training/experiment-config-reference>`.

To learn about this API, you can start by reading the trial definitions from the following examples:

-  :download:`Fashion MNIST example </examples/fashion_mnist_tf_keras.tgz>`
-  :download:`CIFAR-10 example </examples/cifar10_tf_keras.tgz>`

***********
 Load Data
***********

.. note::

   Before loading data, visit :ref:`load-model-data` to understand how to work with different
   sources of data.

Loading data is done by defining :meth:`~determined.keras.TFKerasTrial.build_training_data_loader`
and :meth:`~determined.keras.TFKerasTrial.build_validation_data_loader` methods. Each should return
one of the following data types:

#. A tuple ``(x, y)`` of NumPy arrays. x must be a NumPy array (or array-like), a list of arrays (in
   case the model has multiple inputs), or a dict mapping input names to the corresponding array, if
   the model has named inputs. y should be a numpy array.

#. A tuple ``(x, y, sample_weights)`` of NumPy arrays.

#. A ``tf.data.dataset`` returning a tuple of either (inputs, targets) or (inputs, targets,
   sample_weights).

#. A ``keras.utils.Sequence`` returning a tuple of either (inputs, targets) or (inputs, targets,
   sample weights).

If using ``tf.data.Dataset``, users are required to wrap both their training and validation dataset
using :meth:`self.context.wrap_dataset <determined.keras.TFKerasTrialContext.wrap_dataset>`. This
wrapper is used to shard the dataset for distributed training. For optimal performance, users should
wrap a dataset immediately after creating it.

.. include:: ../../../_shared/note-dtrain-learn-more.txt

******************
 Define the Model
******************

Users are required wrap their model prior to compiling it using :meth:`self.context.wrap_model
<determined.keras.TFKerasTrialContext.wrap_model>`. This is typically done inside
:meth:`~determined.keras.TFKerasTrial.build_model`.

******************************************
 Customize Calling Model Fitting Function
******************************************

The :class:`~determined.keras.TFKerasTrial` interface allows the user to configure how ``model.fit``
is called by calling :meth:`self.context.configure_fit()
<determined.keras.TFKerasTrialContext.configure_fit>`.

***************
 Checkpointing
***************

A checkpoint includes the model definition (Python source code), experiment configuration file,
network architecture, and the values of the model's parameters (i.e., weights) and hyperparameters.
When using a stateful optimizer during training, checkpoints will also include the state of the
optimizer (i.e., learning rate). You can also embed arbitrary metadata in checkpoints via a
:ref:`Python SDK <store-checkpoint-metadata>`.

TensorFlow Keras trials are checkpointed to a file named ``determined-keras-model.h5`` using
``tf.keras.models.save_model``. You can learn more from the `TF Keras docs
<https://www.tensorflow.org/versions/r1.15/api_docs/python/tf/keras/models/save_model>`__.

***********
 Callbacks
***********

To execute arbitrary Python code during the lifecycle of a :class:`~determined.keras.TFKerasTrial`,
implement the :class:`determined.keras.callbacks.Callback` interface (an extension of the
``tf.keras.callbacks.Callbacks`` interface) and supply them to the
:class:`~determined.keras.TFKerasTrial` by implementing
:meth:`~determined.keras.TFKerasTrial.keras_callbacks`.
