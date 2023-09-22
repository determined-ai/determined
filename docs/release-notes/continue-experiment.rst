:orphan:

**New Features**

-  Experiments: Add new CLI command ``det e continue <experiment-id>`` to resume or recover training
   for an experiment. This can be used on experiments that have completed successfully or failed.
   This is limited to single-searcher experiments. If the continue feature is used, it might not be
   possible to replicate results of the continued experiment.
