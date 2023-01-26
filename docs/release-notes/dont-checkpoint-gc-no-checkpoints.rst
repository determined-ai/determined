:orphan:

**Improvements**

-  Experiments: If an experiment with no checkpoints is deleted, a checkpoint GC task will no longer
   be launched. Launching a checkpoint GC task could prevent experiments with certain incorrect
   configuration from being deleted.
