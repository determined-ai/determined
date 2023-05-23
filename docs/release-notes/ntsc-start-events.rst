:orphan:

**Removed Features**

-  CLI: `det notebook|tensorboard start` no longer block for the whole lifecycle of the Notebook or
   Tensorboard procees. These will also not stream related event logs. Users should use the existing
   `det notebook|tensorboard|task logs` commands to stream logs from the process.
