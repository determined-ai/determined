:orphan:

**New Features**

-  Kubernetes: When users submit Determined tasks, the system now launch Kubernetes jobs on their
   behalf of instead of Kubernetes pods. The new system interacts differently with other Kubernetes
   concepts; e.g., Determined tasks will now work properly with Kubernetes resources quotas.

   Because of this, we now require permissions to create, get, list, delete and watch Kubernetes
   jobs resources.
