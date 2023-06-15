:orphan:

**Improvements**

-  Errors: Errors that return 404 or 'Not Found' codes now have standardized messaging using the
   format "<task/trial/workspace etc.> <ID> not found". In addition, if RBAC is enabled, the error
   message includes a suffix to remind users to check their permissions. This is because with RBAC
   enabled, permission denied errors & not found errors would both return a 'Not Found' response.
