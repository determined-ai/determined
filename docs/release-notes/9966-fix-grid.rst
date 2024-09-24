:orphan:

**Fixes**

-  Previously, during a grid search, if a hyperparameter contained an empty nested hyperparameter
   (that is, just an empty map), that hyperparameter would not appear in the hparams passed to the
   trial.
