:orphan:

**Improvements**

-  Trials: Metric storage has been optimized for reading summaries of metrics reported during a
   trial.

   This has the effect of causing upgrades from a previous version to this version or a future
   version to take an extended period of time for clusters to upgrade with a large amount of trials
   and training steps reported. An example database with 10,000 trials with 125 million training
   metrics on a small instance can have 6+ hours of downtime upgrading.

   As an optional mitigation users with large databases can choose to manually run the `this sql
   file
   <https://github.com/determined-ai/determined/blob/main/master/static/migrations/20230425100036_add-summary-metrics.tx.up.sql>`__
   against their cluster's database in advance of a version upgrade while it is still running to
   minimize downtime. This is purely optional and only recommended for sufficiently large databases.
