:orphan:

**New Features**

-  Checkpoints: Add support for deleting a subset of files from checkpoints.

   The CLI command ``det checkpoint rm uuid1,uuuid2 --glob deleteDir1/** --glob deleteDir2`` has
   been added to remove all files in checkpoints specified that match at least one glob provided.

   The SDK method :meth:`determined.experimental.client.Checkpoint.remove_files` has been added to
   delete files matching a list of globs provided.

**Improvements**

-  Checkpoints: Determined previously and currently creates a ``metadata.json`` file used for
   internal purposes in checkpoint storage. New checkpoints will now list a ``metadata.json`` when
   viewing files a checkpoint has in Determined. This file was previously created but hidden from
   checkpoint related views and APIs.
