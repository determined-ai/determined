:orphan:

**Breaking Changes**

-  Cluster: Several unused columns have been dropped from ``raw_steps``, ``raw_validations``,
   ``raw_checkpoints`` tables. The database migration will involve a sequential scan for these
   tables, and it may take significant amount of time, depending on the database size and
   performance.
