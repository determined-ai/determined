.. _hyperparameter-tuning:

#######################
 Hyperparameter Tuning
#######################

Hyperparameter tuning is a common machine learning workflow that involves appropriately configuring
the data, model architecture, and learning algorithm to yield an effective model. Hyperparameter
tuning is a :ref:`challenging problem <topic-guides_hp-tuning-basics-difficulty>` in deep learning
given the potentially large number of hyperparameters to consider.

Determined provides support for hyperparameter search as a first-class workflow that is tightly
integrated with Determined's job scheduler, which allows for efficient execution of state-of-the-art
early-stopping based approaches as well as seamless parallelization of these methods.

An intuitive interface is provided to use hyperparameter searching. This document covers the
following things:

#. Specify a searcher to find effective hyperparameter settings within the predefined ranges.
#. Configure hyperparameter ranges to search.
#. Instrument model code to use hyperparameters from the experiment configuration.
#. Handle Trial Errors and Early Stopping Requests

*********************************
 Specifying the Search Algorithm
*********************************

Determined supports a :ref:`variety of hyperparameter search algorithms <hyperparameter-tuning>`.
Aside from the ``single`` searcher, a searcher runs multiple trials and decides the hyperparameter
values to use in each trial. Every searcher is configured with the name of the validation metric to
optimize (via the ``metric`` field), in addition to other searcher-specific options. For example,
the (`state-of-the-art <https://arxiv.org/pdf/1810.05934.pdf>`_) ``adaptive_asha`` searcher,
suitable for larger experiments with many trials, is configured with the maximum number of trials to
run, the maximum training length allowed per trial, and the maximum number of trials that can be
worked on simultaneously:

.. code:: yaml

   searcher:
     name: "adaptive_asha"
     metric: "validation_loss"
     max_trials: 16
     max_length:
         epochs: 1
     max_concurrent_trials: 8

For details on the supported searchers and their respective configuration options, refer to
:ref:`hyperparameter-tuning`.

That's it! After submitting an experiment, users can easily see the best validation metric observed
across all trials over time in the WebUI. After the experiment has completed, they can view the
hyperparameter values for the best-performing trials and then export the associated model
checkpoints for downstream serving.

.. image:: /assets/images/adaptive-asha-experiment-detail.png

Adaptive Search
===============

Our default recommended search method is `Adaptive (ASHA) <https://arxiv.org/pdf/1810.05934.pdf>`_,
a state-of-the-art early-stopping based technique that speeds up traditional techniques like random
search by periodically abandoning low-performing hyperparameter configurations in a principled
fashion.

:ref:`Adaptive (ASHA) <topic-guides_hp-tuning-det_adaptive-asha>` offers asynchronous search
functionality more suitable for large-scale HP search experiments in the distributed setting.

Other Supported Methods
=======================

Determined also supports other common hyperparameter search algorithms:

#. :ref:`Single <topic-guides_hp-tuning-det_single>` is appropriate for manual hyperparameter
   tuning, as it trains a single hyperparameter configuration.

#. :ref:`Grid <topic-guides_hp-tuning-det_grid>` brute force evaluates all possible hyperparameter
   configurations and returns the best.

#. :ref:`Random <topic-guides_hp-tuning-det_random>` evaluates a set of hyperparameter
   configurations chosen at random and returns the best.

#. :ref:`Population-based training (PBT) <topic-guides_hp-tuning-det_pbt>` begins as random search
   but periodically replaces low-performing hyperparameter configurations with ones *near* the
   high-performing points in the hyperparameter space.

***********************************
 Configuring Hyperparameter Ranges
***********************************

The first step toward automatic hyperparameter tuning is to define the hyperparameter space, e.g.,
by :ref:`listing the decisions <topic-guides_hp-tuning-basics-example-hyperparameters>` that may
impact model performance. For each hyperparameter in the search space, the machine learning engineer
specifies a range of possible values in the experiment configuration:

.. code:: yaml

   hyperparameters:
     ...
     dropout_probability:
       type: double
       minval: 0.2
       maxval: 0.5
     ...

Determined supports the following searchable hyperparameter data types:

#. ``int``: an integer within a range
#. ``double``: a floating point number within a range
#. ``log``: a logarithmically scaled floating point number---users specify a ``base`` and Determined
   searches the space of `exponents` within a range
#. ``categorical``: a variable that can take on a value within a specified set of values---the
   values themselves can be of any type

The :ref:`experiment configuration reference <experiment-configuration_hyperparameters>` details
these data types and their associated options.

**************************
 Instrumenting Model Code
**************************

Determined injects hyperparameters from the experiment configuration into model code via a context
object in the Trial base class. This :class:`~determined.TrialContext` object exposes a
:func:`~determined.TrialContext.get_hparam` method that takes the hyperparameter name. At trial
runtime, Determined injects a value for the hyperparameter. For example, to inject the value of the
``dropout_probability`` hyperparameter defined above into the constructor of a PyTorch `Dropout
<https://pytorch.org/docs/stable/nn.html#dropout>`_ layer:

.. code:: python

   nn.Dropout(p=self.context.get_hparam("dropout_probability"))

To see hyperparameter injection throughout a complete trial implementation, refer to the
:doc:`/training-apis/index`.

***************************************************
 Handling Trial Errors and Early Stopping Requests
***************************************************

When a trial encounters an error or fails unexpectedly, Determined will restart it from the latest
checkpoint unless we have done so :ref:`max_restarts <max-restarts>` times, which is configured in
the experiment configuration. Once we have reached ``max_restarts``, any further trials that fail
will be marked as errored and will not be restarted. For search methods that adapt to validation
metric values (:ref:`Adaptive (ASHA) <topic-guides_hp-tuning-det_adaptive-asha>`, and
:ref:`Population-based training (PBT) <topic-guides_hp-tuning-det_pbt>`), we do not continue
training errored trials, even if the search method would typically call for us to continue training.
This behavior is useful when some parts of the hyperparameter space result in models that cannot be
trained successfully (e.g., the search explores a range of batch sizes and some of those batch sizes
cause GPU OOM errors). An experiment can complete successfully as long as at least one of the trials
within it completes successfully.

Trial code can also request that training be stopped early, e.g., via a framework callback such as
`tf.keras.callbacks.EarlyStopping
<https://www.tensorflow.org/api_docs/python/tf/keras/callbacks/EarlyStopping>`__ or manually by
calling :meth:`determined.TrialContext.set_stop_requested`. When early stopping is requested,
Determined will finish the current training or validation workload and checkpoint the trial. Trials
that are stopped early are considered to be "completed", whereas trials that fail are marked as
"errored".

.. toctree::
   :maxdepth: 1
   :hidden:

   hp-adaptive-asha
   hp-constraints-det
   hp-grid
   hp-pbt
   hp-random
   hp-single
   hp-tuning-defined
