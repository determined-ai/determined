.. _topic-guides_hp-tuning-det_random:

###############
 Random Method
###############

The ``random`` search method generates ``max_trials`` trials with hyperparameters chosen uniformly
at random from the configured hyperparameter space. Each trial is trained for the number of units
specified by ``max_length`` (see :ref:`Training Units <experiment-configuration_training_units>`)
and then then the trial's validation metrics are computed.

See :ref:`Experiment Configuration <experiment-configuration_searcher>`.
