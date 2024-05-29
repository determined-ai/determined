:orphan:

**New Features**

-  Kubernetes: Add a feature where Determined offers the users to provide custom Checkpoint GC pod spec.
      This configuration is done using the ``task_container_defaults.checkpoint_gc_pod_spec`` field
      within your ``value.yaml`` file. When a custom CheckpointGC pod spec is defined, it takes
      precedence over the default CheckpointGC pod specifications. This provides flexibility to
      tailor the garbage collection settings according to the specific GC needs.
