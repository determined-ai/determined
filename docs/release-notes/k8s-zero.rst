:orphan:

**Improvements**

-  Kubernetes: zero-slot tasks on gpu clusters will not request ``nvidia.com/gpu: 0`` resources any
   more, allowing them to be schedule on cpu-only nodes.
