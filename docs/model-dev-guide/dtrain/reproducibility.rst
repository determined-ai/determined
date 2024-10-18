.. _reproducibility:

#################
 Reproducibility
#################

Determined aims to support *reproducible* machine learning experiments: that is, the result of
running a Determined experiment should be deterministic, so that rerunning an experiment produces an
identical model. This ensures that in the event of model loss, recovery is possible by rerunning the
experiment responsible for its creation.

**********
 Progress
**********

While the current version of Determined offers limited support for reproducibility, challenges arise
due to the inherent complexity of the hardware and software stack typically utilized in deep
learning environments.

Determined effectively manages and reproduces several sources of randomness, including:

-  Hyperparameter sampling decisions.
-  The initial weights for a given hyperparameter configuration.
-  Data shuffling during trial training.
-  Utilization of dropout or other random layers.

However, it's important to note that Determined does not currently provide mechanisms for
controlling non-deterministic floating-point operations. Most modern deep learning frameworks employ
floating-point operations that may result in non-deterministic outcomes, particularly on GPUs.
Achieving reproducible results is feasible when training exclusively on CPUs, as elaborated in the
following sections.

**************
 Random Seeds
**************

Each Determined experiment is associated with an **experiment seed**: an integer ranging from 0 to
2\ :sup:`31`--1. The experiment seed can be set using the ``reproducibility.experiment_seed`` field
of the experiment configuration. If an experiment seed is not explicitly specified, the master will
assign one automatically.

The experiment seed is used as a source of randomness for any hyperparameter sampling procedures.
The experiment seed is also used to generate a **trial seed** for every trial associated with the
experiment.

When training on-cluster, the trial seed is accessible via
:class:`det.get_cluster_info().trial.trial_seed <determined.get_cluster_info>`

*******************
 Coding Guidelines
*******************

To achieve reproducible initial conditions in an experiment, please follow these guidelines:

-  Use the `np.random <https://docs.scipy.org/doc/numpy-1.14.0/reference/routines.random.html>`__ or
   `random <https://docs.python.org/3/library/random.html>`__ APIs for random procedures, such as
   shuffling of data. Both PRNGs will be initialized with the trial seed by Determined
   automatically.

-  Use the trial seed to seed any randomized operations (e.g., initializers, dropout) in your
   framework of choice. For example, Keras `initializers <https://keras.io/initializers/>`__ accept
   an optional seed parameter. Again, it is not necessary to set any *graph-level* PRNGs (e.g.,
   TensorFlow's ``tf.set_random_seed``), as Determined manages this for you.

**************************************
 Deterministic Floating Point on CPUs
**************************************

When doing CPU-only training with TensorFlow, it is possible to achieve floating-point
reproducibility throughout optimization:

.. code:: python

   tf.config.threading.set_intra_op_parallelism_threads(1)
   tf.config.threading.set_inter_op_parallelism_threads(1)

.. warning::

   Disabling thread parallelism may negatively affect performance. Only enable this feature if you
   understand and accept this trade-off.

*********************
 Pausing Experiments
*********************

TensorFlow does not fully support the extraction or restoration of a single, global RNG state.
Consequently, pausing experiments that use a TensorFlow-based framework may introduce an additional
source of entropy.
