:orphan:

**Bug Fixes**

-  Kubernetes: Fix an issue where Determined failed to report slots as occupied when non Determined
   jobs were running on namespaces besides 'default'. For Determined to detect non Determined jobs
   they must be running in a namespace that Determined can launch jobs into.
