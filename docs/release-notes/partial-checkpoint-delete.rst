:orphan:

**New Features**

-  Checkpoints: Add support for deleting a subset of files from checkpoints.

   The CLI command ``det checkpoint rm uuid1,uuuid2 --glob deleteDir1/** --glob deleteDir2`` has
   been added to remove all files in checkpoints specified that match at least one glob provided.

   The SDK method :meth:`determined.experimental.client.Checkpoint.remove_files` has been added to
   delete files matching a list of globs provided.
