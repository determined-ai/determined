:orphan:

**Deprecations**

-  Cluster: Amazon Aurora V1 will reach End of Life at the end of 2024, and will no longer be the default
    persistent storage for AWS Determined deployments at that point in time. Users are encouraged to migrate 
    to Amazon RDS for PostgreSQL. 
-  Cluster: ``det deploy aws`` in Determined 0.37.0 and later will no longer support Amazon Aurora V1 as the 
    default persistent storage, instead defaulting to the ``simple-rds`` deployment type.

-  Database: Postgres 12 will reach End of Life on November 14, 2024. Determined instances using a Postgres 12 or 
    earlier versions are encouraged to upgrade to Postgres 13 or later to ensure continued support.
