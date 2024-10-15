.. _api-keras-ug:

###########
 Keras API
###########

.. meta::
   :description: Learn how to use the Keras API to train a Keras model. This user guide walks you through loading your data, defining the model, customizing how the model.fit function is called, checkpointing, and callbacks.

In this guide, you'll learn how to use Determined's ``keras.DeterminedCallback`` while training your
keras model.

+---------------------------------------------------------------------+
| Visit the API reference                                             |
+=====================================================================+
| :ref:`keras-reference`                                              |
+---------------------------------------------------------------------+

This document guides you through training a Keras model in Determined.  You will need to update your
``model.fit()`` call to include a :class:`~determined.keras.DeterminedCallback` and submit it to
a Determined cluster.

To learn about this API, you can start by reading the ``train.py`` script in the `Iris
categorization example
<https://github.com/determined-ai/determined-examples/tree/main/computer_vision/iris_tf_keras>`__.

**********************
 Configure Entrypoint
**********************

Determined requires you to launch training jobs by submitting them with an
:ref:`experiment-configuration`, which tells the Determined master how to start your container.  For
Keras training, you should always wrap your training script in Determined's :ref:`TensorFlow
launcher <launch-tensorflow>`:

.. code:: yaml

   entrypoint: >-
     python3 -m determined.launch.tensorflow --
     python3 my_train.py --my-arg...

Determined's TensorFlow launcher will automatically configure your training script with the right
``TF_CONFIG`` environment variable for distributed training when distributed resources are
available, and will safely do nothing when they are not.

****************************************************************
 Obtain a ``det.core.Context`` and a ``tf.distribute.Strategy``
****************************************************************

When using distributed training, TensorFlow requires you to create your ``Strategy`` early in the
process lifetime, before creating your model.

Since you wrapped your training script in Determined's TensorFlow launcher, you can use Determined's
``core.DistributedContext.from_tf_config()`` helper, which will create both a suitable
``DistributedContext`` and ``Strategy`` for the training environment in your training job.  Then you
can feed that ``DistributedContext`` to ``det.core.init()`` to get a ``core.Context``, and feed all
of that to your ``main()`` function (or equivalent) in your training script:

.. code:: python

   if __name__ == "__main__":
       distributed, strategy = det.core.DistributedContext.from_tf_config()
       with det.core.init(distributed=distributed) as core_context:
           main(core_context, strategy)

*****************
 Build the Model
*****************

Building a distributed-capable model is easy in keras; you just need to wrap your model building and
compiling in the ``strategy.scope()``.  See the `TensorFlow documentation
<https://www.tensorflow.org/tutorials/distribute/keras
#create_the_model_and_instantiate_the_optimizer>`__ for more detail.

.. code:: python

   def main(core_context, strategy):
       with strategy.scope():
           model = my_build_model()
           model.compile(...)

***********************************
 Create the ``DeterminedCallback``
***********************************

The :class:`~determined.keras.DeterminedCallback` automatically integrates your training with the
Determined cluster. It reports both train and test metrics, reports progress, saves checkpoints, and
uploads them to checkpoint storage. Additionally, it manages preemption signals from the Determined
master (for example, when you pause your experiment), gracefully halting training and later resuming
from where it left off.

The :class:`~determined.keras.DeterminedCallback` will automatically integrate your training with
the Determined cluster.  It reports train and test metrics, it reports progress, it saves
checkpoints, and it uploads them to checkpoint storage.  It also handles preemption signals from the
Determined master (such as if you pause your experiment), shutting down training, then it restores
training from where it left off when the experiment continues.

The ``DeterminedCallback`` has only three required inputs:
   -  the ``core_context`` you already created
   -  a ``checkpoint`` UUID to start training from, or ``None``
   -  a ``continue_id`` used to decide how to treat the checkpoint

In training jobs, an easy value for ``checkpoint`` is ``det.get_cluster_info().latest_checkpoint``,
which will automatically be populated with the latest checkpoint saved by this trial, or ``None``.
If, for example, you wanted to start training from a checkpoint and support pausing and resuming,
you could use ``info.latest_checkpoint or my_starting_checkpoint``.

The ``continue_id`` helps the ``DeterminedCallback`` decide if the provided checkpoint represents
just the starting weights and training should begin at epoch=0, or if the checkpoint represents a
partially complete training that should pick up where it left off (at epoch > 0).  The provided
``continue_id`` is saved along with every checkpoint, and when loading the starting checkpoint, if
the ``continue_id`` matches what was in the checkpoint, training state is also loaded from the
checkpoint.  In training jobs, an easy value for ``continue_id`` is
``det.get_cluster_info.trial.trial_id``.

See the reference for :class:`~determined.keras.DeterminedCallback` for details on its optional
parameters.

.. code:: python

   info = det.get_cluster_info()
   assert info and info.task_type == "TRIAL", "this example only runs as a trial on the cluster"

   det_cb = det.keras.DeterminedCallback(
       core_context,
       checkpoint=info.latest_checkpoint,
       continue_id=info.trial.trial_id,
    )

***********
 Load Data
***********

Loading data is done as usual, though additional considerations may arise if your existing
data-loading code is not container-ready. For more details, see :ref:`load-model-data`.

If you want to take advantage Determined's distributed training, you may need to ensure that
your input data is properly sharded.  See `TensorFlow documentation
<https://www.tensorflow.org/tutorials/distribute/input#sharding>`__ for details.

.. include:: ../../../_shared/note-dtrain-learn-more.txt

*************************
 TensorBoard Integration
*************************

Optionally, you can use Determined's :class:`~determined.keras.TensorBoard` callback, which extends
keras' ``TensorBoard`` callback with the ability to automatically upload metrics to Determined's
checkpoint storage.  Determined's ``TensorBoard`` callback is configured identically to keras'
except it takes an additional ``core_context`` initial argument:

.. code:: python

   tb_cb = det.keras.TensorBoard(core_context, ...)

Then simply include it in your ``model.fit()`` as normal.

*************************
 Calling ``model.fit()``
*************************

The only remaining step is to pass your callbacks to your ``model.fit()``:

.. code:: python

   model.fit(
       ...,
       callbacks=[det_cb, tb_cb],
   )
