:orphan:

**Bug Fixes**

-  Kubernetes: Fix an issue where task's submitted with custom pod specs would have the custom
   nodeAffinity ignored.

   Upgrading from a version before this feature to a version after this feature only on Kubernetes
   can cause queued allocations with a custom pod spec nodeAffinity to be killed. Users can pause
   queued experiments to avoid this.
