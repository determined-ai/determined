:orphan:

**Bug Fixes**

-  Hyperparameter Search: Prevent hyperparameter searches from incorrectly terminating early when
   starting a new trial in response to the last previously open trial closing. One common way for
   this situation to arise is when running an experiment with ``max_concurrent_trials`` set to
   ``1``.
