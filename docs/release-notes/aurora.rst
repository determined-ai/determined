:orphan:

**Bug Fixes**

-  Since 0.28.1, all deployments using Amazon Aurora PostgreSQL-Compatible Edition Serverless V1 for
   its database were at risk of becoming completely unresponsive whenever Aurora hits certain
   autoscaling errors. Multiple ``det deploy aws`` deployment types (default ``simple``, ``vpc``,
   ``efs``, ``fsx``, ``secure``) are affected. AWS installations using RDS were not affected,
   including ``det deploy aws --deployment-type=govcloud`` deployments. We recommend all users
   running affected setups to upgrade as soon as possible.
