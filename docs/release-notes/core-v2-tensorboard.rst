:orphan:

**Bug Fixes**

-  Core API: On context closure, properly save all TensorBoard files not related to metrics
   reporting, particularly the native profiler traces.
-  Core API v2: Fix an issue where TensorBoard files were not saved for managed experiments.
