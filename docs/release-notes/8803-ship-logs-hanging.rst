:orphan:

**Fixes**

-  Since 0.26.2, it was possible to cause Determined Trials and Commands to hang after the main
   process exited but before the container exited, by starting a non-terminating subprocess from
   your training script or command that kept an open stdout or stderr file descriptor. Now, logs
   from subprocesses of your main process are ignored after your main process has exited.
