:orphan:

**Improvement**

-  Avoid including RBAC permission granted audit logs with other master logs. These logs are no
   longer exposed through master logs API but are still pushed out through master's stdout.
