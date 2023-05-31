:orphan:

**Breaking Changes**

-  Kubernetes: When a pod spec is specified in both ``task_container_defaults`` and in the
   experiment/job configuration, the pod spec is merged according to `strategic merge patch
   <https://kubernetes.io/docs/tasks/manage-kubernetes-objects/update-api-object-kubectl-patch/#use-a-strategic-merge-patch-to-update-a-deployment>`__.
   The previous behavior was using only the experiment/job configuration if supplied.
