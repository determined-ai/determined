:orphan:

**Breaking Changes**

-  Kubernetes: When a pod spec is specified in both ``task_container_defaults`` and in experiment /
   job configuration the pod spec is merged according to `strategic merge patch
   <https://github.com/kubernetes/community/blob/master/contributors/devel/sig-api-machinery/strategic-merge-patch.md>`__.
   The previous behavior was only using experiment / job configuration if supplied.
