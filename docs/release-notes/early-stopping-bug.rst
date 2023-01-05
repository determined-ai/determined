:orphan:

**Bug Fixes**

-  Distributed training: Fix a bug where a distributed training trial that calls
   ``context.set_stop_requested`` caused the trial to error instead of successfully complete.
