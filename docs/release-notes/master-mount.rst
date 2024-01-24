:orphan:

**Improvements**

-  Helm: Add support for Downloading Checkpoints when using ``shared_fs``. Adds a ``mountToServer``
   value under ``checkpointStorage``. By default, this parameter is set to ``false`` preserving the
   current behavior. However when it's set to ``true`` and the storage type is ``shared_fs`` it
   enables the hostpath mount on the server. This allows for the use of ``checkpoint.download()`` to
   work with ``shared_fs`` on Determined starting from version ``0.27.0`` and later.
