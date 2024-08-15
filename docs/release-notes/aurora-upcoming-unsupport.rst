:orphan:

**Deprecations**

-  Cluster: Amazon Aurora V1 will reach End of Life at the end of 2024 and will no longer be the default
      persistent storage for AWS Determined deployments. Users should migrate to Amazon RDS for
      PostgreSQL.

-  Cluster: After Amazon Aurora V1 reaches End of Life, support for Amazon Aurora V1 in ``det deploy aws`` will be
      removed. The deployment will default to the ``simple-rds`` type, which uses Amazon RDS.

-  Database: Postgres 12 will reach End of Life on November 14, 2024. Determined instances using Postgres 12 or earlier
      should upgrade to Postgres 13 or later to ensure continued support.
