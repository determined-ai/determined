:orphan:

**Improvements**

-  Errors: Errors that return 404 or 'Not Found' codes are now standardized in their messageas
   "<task/trial/workspace etc.> <ID> not found". Additionally, when RBAC is enabled, the errors adds
   a suffix to remind users to check their permissions, since, with RBAC, permission denied errors &
   not found errors would both return as 'Not Found'.
