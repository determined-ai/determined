:orphan:

**Bug Fixes**

-  API/Tasks: Fix bug where a master-configured ``log_retention_days`` value is not applied to
   experiments and tasks. Now, newly created experiments will apply the master-configured value, and
   all pre-existing experiments will be subjected to the configured ``log_retention_days``.
