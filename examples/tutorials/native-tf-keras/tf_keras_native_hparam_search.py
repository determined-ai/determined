"""
.. _tutorials_native-api-hparam-search:

Native API: Hyperparameter Searches
===================================

One powerful application of the Native API is that it can be used to seamlessly
launch parallel hyperparameter searches with a minimal set of code changes.
This example builds on top of :ref:`tutorials_native-api-basics` to demonstrate
this.

"""
import tensorflow as tf
import determined as det
from determined.experimental.keras import init

config = {
    "searcher": {"name": "adaptive_simple", "metric": "val_loss", "max_steps": 5, "max_trials": 5},
    "hyperparameters": {
        "num_units": det.Integer(64, 256),
        "dropout": det.Double(0.0, 0.5),
        "activation": det.Categorical(["relu", "tanh", "sigmoid"]),
        "global_batch_size": 32,
    },
}

###############################################################################
#
# First, configure the ``hyperparameters`` field in the experiment configuration
# to set up a hyperparameter search space.  :py:func:`determined.Integer`,
# :py:func:`determined.Double`, :py:func:`determined.Categorical`, and
# :py:func:`determined.Log` are utility functions for specifying distributions.
#
# Next, use the ``searcher`` field to configure the desired hyperparameter
# search algorithm. In this case, we're configuring a :ref:`simple adaptive
# search <topic-guides_hp-tuning-det_adaptive-simple>` to optimize over five
# possible choices of hyperparameters. See :ref:`topic-guides_hp-tuning-det` for
# a full list of available hyperparameter tuning algorithms.

(x_train, y_train), (x_test, y_test) = tf.keras.datasets.mnist.load_data()
x_train, x_test = x_train / 255.0, x_test / 255.0

# When running this code from a notebook, add a `command` argument to init()
# specifying the notebook file name.
context = init(config, context_dir=".")
model = tf.keras.models.Sequential(
    [
        tf.keras.layers.Flatten(input_shape=(28, 28)),
        tf.keras.layers.Dense(
            context.get_hparam("num_units"), activation=context.get_hparam("activation")
        ),
        tf.keras.layers.Dropout(context.get_hparam("dropout")),
        tf.keras.layers.Dense(10, activation="softmax"),
    ]
)
model = context.wrap_model(model)
model.compile(
    tf.keras.optimizers.Adam(name='Adam'),
    loss="sparse_categorical_crossentropy", metrics=["accuracy"])
model.fit(x_train, y_train, validation_data=(x_test, y_test), epochs=5)

###############################################################################
#
# Now that you've configured your hyperparameter ranges, you can use
# :py:func:`determined.keras.TFKerasNativeContext.get_hparam` anywhere in model
# code to plug the hyperparameter value into your training logic.  Determined
# will manage initializing the ``context`` with a unique set of hyperparameter
# values for every trial executed on the cluster.
#
# Reference
# ---------
#
# * :ref:`hyperparameter-tuning`
# * :ref:`experiment-configuration`
