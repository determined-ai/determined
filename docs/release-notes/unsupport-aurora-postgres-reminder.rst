:orphan:

**Deprecations**

-  Cluster: A reminder that Amazon Aurora V1 will reach End of Life at the end of 2024. It is no
   longer supported as the default persistent storage for AWS Determined deployments. We recommend
   that users migrate to Amazon RDS for PostgreSQL. For more information, visit the `migration
   instructions <https://gist.github.com/maxrussell/c67f4f7d586d55c4eb2658cc2dd1c290>`_.

-  Cluster: After Amazon Aurora V1 reaches End of Life, support for Amazon Aurora V1 in ``det deploy
   aws`` will be removed. Future deployments will default to the ``simple-rds`` type, which uses
   Amazon RDS for PostgreSQL. Changes to the deployment code will ensure this transition to the new
   default.

-  Database: As a follow-up to the earlier notice, PostgreSQL 12 will reach End of Life on November
   14, 2024. Instances still using PostgreSQL 12 or earlier should upgrade to PostgreSQL 13 or later
   to maintain compatibility. The application will log a warning if it detects a connection to any
   PostgreSQL version older than 12, and this warning will be updated to include PostgreSQL 12 once
   it is End of Life.
