.. _topic-guides_hp-tuning-det_single:

######################
 Single Search Method
######################

The ``single`` search method does a very minimal "search": it trains a single hyperparameter
configuration for the number of units specified by ``max_length`` (see :ref:`Training Units
<experiment-configuration_training_units>`) and then performs validation. This method is useful for
testing or for training a single model configuration until convergence.

See :ref:`Experiment Configuration <experiment-configuration_searcher>`.
