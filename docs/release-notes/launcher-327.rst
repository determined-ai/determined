:orphan:

**Improvements**

-  Cluster: HPC Launcher 3.2.7 migrates the ``resource_manager.job_storage_root`` to a more
   efficient format. This happens automatically, but once migrated you cannot downgrade to an older
   version of the HPC launcher.

-  Cluster: The ``manage-singularity-cache`` script has added the ``--docker-login`` option to
   enable access to private docker images.
