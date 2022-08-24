:orphan:

**Bug Fixes**

-  Hyperparameter Search: Prevent the random and grid hyperparameter searches from spawning more
   trials than allowed by ``max_concurrent_trials`` in the event of trial failures.
