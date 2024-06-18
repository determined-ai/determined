:orphan:

**New Features**

-  Kubernetes Configuration: Allow Cluster administrators to define Determined resource pools on
   Kubernetes using node selectors and/or affinities. Configure these settings at the default pod
   spec level under ``task_container_defaults.cpu_pod_spec`` or
   ``task_container_defaults.gpu_pod_spec``. This allows a single cluster to be divided into
   multiple resource pools using node labels.

-  WebUI: Allow resource pool slot counts to reflect the state of the entire cluster. Allow slot
   counts and scheduling to respect node selectors and affinities. This impacts Determined clusters
   deployed on Kubernetes with multiple resource pools defined in terms of node selectors and/or
   affinities.
