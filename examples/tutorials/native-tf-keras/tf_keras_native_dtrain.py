"""
.. _tutorials_native-api-dtrain:

Native API: Distributed Training
================================

One powerful application of the Native API is that it can be used to seamlessly
launch distributed training jobs (both single and multi instance) with a
minimal set of code changes. This example builds on top of
:ref:`tutorials_native-api-basics` to demonstrate this.

"""
import tensorflow as tf
import determined as det
from determined.experimental.keras import init

config = {
    "searcher": {"name": "single", "metric": "val_accuracy", "max_steps": 5},
    "hyperparameters": {"global_batch_size": "256"},
    "resources": {"slots_per_trial": 8},
}

###############################################################################
#
# First, configure the ``resources.slots_per_trial`` field in the experiment
# configuration to choose the number of :ref:`slots<terminology-concepts>` to
# train on. You should ensure that the Determined cluster you're using to launch
# the experiment has a sufficient amount of slots available. In the example
# above, we have configured the experiment to use 8 slots (GPUs) to train a
# single model in parallel.
#
# In this case, we've configured our experiment to use a ``global_batch_size``
# of 256 across all slots, or a sub-batch size of 32 on each slot.

(x_train, y_train), (x_test, y_test) = tf.keras.datasets.mnist.load_data()
x_train, x_test = x_train / 255.0, x_test / 255.0

# When running this code from a notebook, add a `command` argument to init()
# specifying the notebook file name.
context = init(config, context_dir=".")
model = tf.keras.models.Sequential(
    [
        tf.keras.layers.Flatten(input_shape=(28, 28)),
        tf.keras.layers.Dense(128, activation="relu"),
        tf.keras.layers.Dropout(0.2),
        tf.keras.layers.Dense(10, activation="softmax"),
    ]
)
model = context.wrap_model(model)
model.compile(
    optimizer=tf.keras.optimizers.Adam(name='Adam'), 
    loss="sparse_categorical_crossentropy", metrics=["accuracy"])
model.fit(x_train, y_train, validation_data=(x_test, y_test), epochs=5)

###############################################################################
#
# Now, configure and launch the training job as done in
# :ref:`tutorials_native-api-basics`. Note that no code changes are required to
# scale up to distributed training.
#
# We use
# :py:func:`~determined.keras.TFKerasNativeContext.get_per_slot_batch_size()` to
# set the framework ``batch_size`` argument. Determined will handle initializing
# the context of each distributed training worker such that it's sub-batch size
# is returned by this function. Because Determined manipulates the batch size
# as a first-class configuration property, ``global_batch_size`` is a required
# hyperparameter in all experiments.
#
# Reference
# ---------
#
# * :ref:`multi-gpu-training`
# * :ref:`experiment-configuration`
