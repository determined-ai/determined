.. _experiment:

###########
 Experiment
###########

We use **experiments** to represent the basic unit of running the model training code. An experiment
is a collection of one or more trials that are exploring a user-defined hyperparameter space. For
example, during a learning rate hyperparameter search, an experiment might consist of three trials
with learning rates of .001, .01, and .1.

.. _concept-trial:

A **trial** is a training task with a defined set of hyperparameters. A common degenerate case is an
experiment with a single trial, which corresponds to training a single deep learning model.

In order to run experiments, you need to write your model training code. We use **model definition**
to represent a specification of a deep learning model and its training procedure. It contains
training code that implements :doc:`training APIs </training/apis-howto/overview>`.

For each experiment, you can configure a **searcher**, also known as a **search algorithm**. The
search algorithm determines how many trials will be run for a particular experiment and how the
hyperparameters will be set. More information can be found at :doc:`/training/hyperparameter/overview`.
