:orphan:

**Breaking Changes**

-  Database: Several unused columns have been dropped from ``raw_steps``, ``raw_validations``,
   ``raw_checkpoints`` database tables. The database migration will involve a sequential scan for
   these tables, and it may take significant amount of time, depending on the database size and
   performance.
