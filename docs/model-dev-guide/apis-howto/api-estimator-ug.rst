###############
 Estimator API
###############

.. meta::
   :description: EstimatorTrial is deprecated.

.. warning::

   ``EstimatorTrial`` is deprecated and will be removed in a future version. TensorFlow has advised
   Estimator users to switch to Keras since TensorFlow 2.0 was released. Consequently, we recommend
   users of EstimatorTrial to switch to the :class:`~determined.keras.TFKerasTrial` class.

In this guide, you'll learn how to use the Estimator API.

+-----------------------------------------------------------------------+
| Visit the API reference                                               |
+=======================================================================+
| :doc:`/reference/training/api-estimator-reference`                    |
+-----------------------------------------------------------------------+

This document guides you through training a Estimator model in Determined. You need to implement a
trial class that inherits :class:`~determined.estimator.EstimatorTrial` and specify it as the
entrypoint in the :doc:`experiment configuration </reference/training/experiment-config-reference>`.

*******************************
 Define Optimizer and Datasets
*******************************

.. note::

   Before loading data, read this document :doc:`/model-dev-guide/load-model-data` to understand how
   to work with different sources of data.

To use ``tf.estimator`` models with Determined, you'll need to wrap your optimizer and datasets
using :meth:`~determined.estimator.EstimatorTrialContext.wrap_optimizer` and
:meth:`~determined.estimator.EstimatorTrialContext.wrap_dataset`. Note that the concrete context
object where these functions will be found will be in
:class:`determined.estimator.EstimatorTrialContext`.

.. _estimators-custom-reducers:

****************
 Reduce Metrics
****************

Determined supports proper reduction of arbitrary validation metrics during distributed training by
allowing users to define custom reducers for their metrics. Custom reducers can be either a function
or an implementation of the :class:`determined.estimator.MetricReducer` interface.

See :func:`context.make_metric() <determined.estimator.EstimatorTrialContext.make_metric>` for more
details.

***************
 Checkpointing
***************

A checkpoint includes the model definition (Python source code), experiment configuration file,
network architecture, and the values of the model's parameters (i.e., weights) and hyperparameters.
When using a stateful optimizer during training, checkpoints will also include the state of the
optimizer (i.e., learning rate). You can also embed arbitrary metadata in checkpoints via the
:ref:`Python SDK <store-checkpoint-metadata>`.

TensorFlow Estimator trials are checkpointed using the `SavedModel
<https://www.tensorflow.org/guide/saved_model>`__ format. Please consult the TensorFlow
documentation for details on how to restore models from the SavedModel format.

***********
 Callbacks
***********

To execute arbitrary Python code during the lifecycle of a ``EstimatorTrial``,
:class:`~determined.estimator.RunHook` extends `tf.estimator.SessionRunHook
<https://www.tensorflow.org/api_docs/python/tf/estimator/SessionRunHook/>`_. When utilizing
:class:`determined.estimator.RunHook`, users can use native estimator hooks such as ``before_run()``
and Determined hooks such as ``on_checkpoint_end()``.

Example usage of :class:`determined.estimator.RunHook` which adds custom metadata checkpoints:

.. code:: python

   class MyHook(determined.estimator.RunHook):
       def __init__(self, context, metadata) -> None:
           self._context = context
           self._metadata = metadata

       def on_checkpoint_end(self, checkpoint_dir) -> None:
           with open(os.path.join(checkpoint_dir, "metadata.txt"), "w") as fp:
               fp.write(self._metadata)


   class MyEstimatorTrial(determined.estimator.EstimatorTrial):
       ...

       def build_train_spec(self) -> tf.estimator.TrainSpec:
           return tf.estimator.TrainSpec(
               make_input_fn(),
               hooks=[MyHook(self.context, "my_metadata")],
           )
