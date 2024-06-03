:orphan:

**New Features**

-  Kubernetes: The system now launches Kubernetes jobs on behalf of users when they submit workloads
   to Determined, instead of launching Kubernetes pods. This change allows Determined to work
   properly with other Kubernetes features like resource quotas.

   As a result, permissions are now required to create, get, list, delete, and watch Kubernetes job
   resources.
