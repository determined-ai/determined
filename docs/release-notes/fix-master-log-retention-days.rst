:orphan:

**Bug Fixes**

-  API/Tasks: Fix a bug where a master-configured ``log_retention_days`` value is not applied to
   experiments and tasks. Now, the master-configured value is correctly applied to new experiments,
   and all pre-existing experiments will also follow the specified ``log_retention_days``.
