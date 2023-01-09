:orphan:

**Bug Fixes**

-  Distributed training: We fixed a bug where a distributed training trial that calls
   context.set_stop_requested was causing the trial to error and preventing it from completing
   successfully.
