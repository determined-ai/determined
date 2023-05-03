:orphan:

**Improvements**

-  Trials: Metric storage has been optimized for reading summaries of metrics reported during a
   trial.

   Extended downtime may result when upgrading from a previous version to this version or a later
   version. This will occur when your cluster contains a large number of trials and training steps
   reported. For example, a database with 10,000 trials with 125 million training metrics on a small
   instance may experience 6 or more hours of downtime during the upgrade.

   (Optional) To minimize downtime, users with large databases can choose to manually run `this sql
   file
   <https://github.com/determined-ai/determined/blob/main/master/static/migrations/20230425100036_add-summary-metrics.tx.up.sql>`__
   against their cluster's database while it is still running before upgrading to a new version.
   This is an optional step and is only recommended for significantly large databases.
