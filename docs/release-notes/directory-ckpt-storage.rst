:orphan:

**New Features**

-  Add new ``directory`` checkpoint storage type, which allows for storing checkpoint and
   TensorBoard data at a specified path inside the task containers. Users are responsible for
   mounting a persistent storage at this path, e.g., a shared PVC using ``pod_spec`` configuration
   in kubernetes-based setups.
