"""
.. _tutorials_native-api-basics:

Native API: Basics
==================

First, let's consider what it looks like to train a very simple model on MNIST
using ``tf.keras``, taken directly from `TensorFlow documentation
<https://www.tensorflow.org/overview>`_.
"""

import tensorflow as tf

(x_train, y_train), (x_test, y_test) = tf.keras.datasets.mnist.load_data()
x_train, x_test = x_train / 255.0, x_test / 255.0

model = tf.keras.models.Sequential(
    [
        tf.keras.layers.Flatten(input_shape=(28, 28)),
        tf.keras.layers.Dense(128, activation="relu"),
        tf.keras.layers.Dropout(0.2),
        tf.keras.layers.Dense(10, activation="softmax"),
    ]
)
model.compile(
    tf.keras.optimizers.Adam(name='Adam'), 
    loss="sparse_categorical_crossentropy", metrics=["accuracy"])
model.fit(x_train, y_train, validation_data=(x_test, y_test), epochs=1)

###############################################################################
#
# Here is what it looks like to train the exact same model using the Native API
# to launch an experiment on a Determined cluster.

import determined as det
from determined import experimental
from determined.experimental.keras import init

config = {
    "searcher": {"name": "single", "metric": "val_acc", "max_steps": 5},
    "hyperparameters": {"global_batch_size": 32},
}

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
    tf.keras.optimizers.Adam(name='Adam'), 
    loss="sparse_categorical_crossentropy", metrics=["accuracy"])
model.fit(x_train, y_train, validation_data=(x_test, y_test), epochs=5)

###############################################################################
#
# Paste the code above into a Python file named ``tf_keras_native.py`` and run
# it as a Python script.
#
# .. note::
#
#       Before submitting any experiments using the Native API, make sure the
#       :ref:`DET_MASTER environment variable is configured to connect to the
#       appropriate IP address <install-cli>`.
#
# .. code:: bash
#
#     $ python tf_keras_native.py
#
# You can also use any environment that supports Python to launch an experiment
# with this code, such as a Jupyter notebook or an IDE.
#
# Let's walk through some of the concepts introduced by the Native API.
#
# Configuration
# -------------

config = {
    "searcher": {"name": "single", "metric": "val_acc", "max_steps": 5},
    "hyperparameters": {"global_batch_size": 16},
}

###############################################################################
#
# Configuring any experiment for use with Determined requires an
# :ref:`experiment-configuration`. In the Native API, this is represented as a
# Python dictionary. There are two *required* fields for every configuration
# submitted via the Native API:
#
# ``searcher``:
#       This field describes how many different :ref:`Trials <concept-trial>`
#       (models) should be trained.  In this case, we've specified to
#       train a ``"single"`` model for five :ref:`training steps
#       <concept-step>`.
# ``hyperparameters``:
#       This field describes the hyperparameters used. ``global_batch_size`` is
#       a required hyperparameter for every experiment -- we'll revisit this
#       requirement in :ref:`tutorials_native-api-dtrain`.
#
# Context
# -------
#
# .. code:: python
#
#     context = init(config, local=False, test=False, context_dir=".")
#
# :ref:`keras-init` is the function that initializes the Determined training
# context. We can think of it as the moment in the training script where
# Determined will "assume control" of the execution of your code. It has two
# three in addition to the configuration:
#
# ``local`` (``bool``):
#       ``local=False`` will submit the experiment to a Determined cluster.
#       ``local=True`` will execute the training loop in your local Python
#       environment (although currently, local training is not implemented, so
#       you must also set ``test=True``). Defaults to False.
#
# ``test`` (``bool``):
#       ``test=True`` will execute a minimal training loop rather than a full
#       experiment. This can be useful for porting or debugging a model because
#       many common errors will surface quickly. Defaults to False.
#
# ``context_dir`` (``str``):
#       Specifies the location of the code you want submitted to the cluster.
#       This is required by Determined to execute your training script in a
#       remote environment (``local=False``). In the common case, "." submits
#       your entire working directory to the Determined cluster.
#
# Wrap Model (``tf.keras`` only)
# ------------------------------
#
# .. code:: python
#
#     model = context.wrap_model(model)
#
# In the case of ``tf.keras``, we will need to use the ``wrap_model`` API to
# make the Determined context aware of the model we want to train with. After
# calling ``wrap_model``, we proceed with the ``compile()`` and ``fit()``
# interfaces defined by TensorFlow to begin training our model remotely.
#
# Next Steps
# ----------
#
# * :ref:`tutorials_native-api-hparam-search`
# * :ref:`tutorials_native-api-dtrain`
