:orphan:

**Improvements**

-  Helm: Add support for Downloading Checkpoints when using ``shared_fs``. Adds a ``mountToMaster``
   value under ``checkpointStorage``. Set to ``false`` by default means no change to current
   behavior. If it's set to ``true`` and the type is ``shared_fs`` it will add that hostpath mount
   to the master as well. This allows for ``checkpoint.download()`` to work with ``shared_fs`` on
   Kubernetes on version ``0.27.0`` and up.
