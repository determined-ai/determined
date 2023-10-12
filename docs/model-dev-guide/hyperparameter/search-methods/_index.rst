.. _search-methods:

################
 Search Methods
################

Determined supports a :ref:`variety of hyperparameter search algorithms <hyperparameter-tuning>`.
Aside from the ``single`` searcher, a searcher runs multiple trials and decides the hyperparameter
values to use in each trial. Every searcher is configured with the name of the validation metric to
optimize (via the ``metric`` field), in addition to other searcher-specific options. For example,
the ``adaptive_asha`` searcher (`arXiv:1810.0593 <https://arxiv.org/pdf/1810.05934.pdf>`_), suitable
for larger experiments with many trials, is configured with the maximum number of trials to run, the
maximum training length allowed per trial, and the maximum number of trials that can be worked on
simultaneously:

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

That's it! After submitting an experiment, you can easily see the best validation metric observed
across all trials over time in the WebUI. After the experiment has completed, you can view the
hyperparameter values for the best-performing trials and then export the associated model
checkpoints for downstream serving.

.. image:: /assets/images/adaptive-asha-experiment-detail.png
   :alt: You can use the Determined WebUI to see the best validation metric observed across all trials over time.

*****************
 Adaptive Search
*****************

Our default recommended search method is `Adaptive (ASHA) <https://arxiv.org/pdf/1810.05934.pdf>`_,
a state-of-the-art early-stopping based technique that speeds up traditional techniques like random
search by periodically abandoning low-performing hyperparameter configurations in a principled
fashion.

:ref:`Adaptive (ASHA) <topic-guides_hp-tuning-det_adaptive-asha>` offers asynchronous search
functionality more suitable for large-scale HP search experiments in the distributed setting.

********************************
 Other Supported Search Methods
********************************

Determined also supports other common hyperparameter search algorithms:

-  :ref:`Single <topic-guides_hp-tuning-det_single>` is appropriate for manual hyperparameter
   tuning, as it trains a single hyperparameter configuration.
-  :ref:`Grid <topic-guides_hp-tuning-det_grid>` evaluates all possible hyperparameter
   configurations by brute force and returns the best.
-  :ref:`Random <topic-guides_hp-tuning-det_random>` evaluates a set of hyperparameter
   configurations chosen at random and returns the best.

You can also implement your own :ref:`custom search methods <topic-guides_hp-tuning-det_custom>`.

.. toctree::
   :maxdepth: 2
   :glob:

   ./*
