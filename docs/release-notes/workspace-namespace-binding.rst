:orphan:

**Breaking Changes**

-  **Cluster:** The ``kubernetes_namespace`` field in the resource pool configuration is no longer
   supported. Users can now submit workloads to specific namespaces by binding workspaces to
   namespaces using the CLI or API.

**New Features**

-  **Cluster:** The ``namespace`` field in the Kubernetes Resource Manager configuration has been
   deprecated. Add a new field, ``default_namespace``. This field serves as the default namespace
   for deploying namespaced resources when the workspace associated with a workload is not bound to
   a specific namespace. The master configuration will accept either ``namespace`` or
   ``default_namespace`` fields; however, providing both fields will result in an error.
