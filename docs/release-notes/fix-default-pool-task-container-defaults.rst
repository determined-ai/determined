:orphan:

**Bug Fixes**

-  Fix an issue where ``task_container_defaults`` for the default resource pools where not respected
   for the experiments and tasks unless they specified the resource pool name explicitly, which has
   been introduced in 0.19.9.
