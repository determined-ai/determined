:orphan:

**New Features**

-  Kubernetes: Add a feature where Determined offers the users to provide custom Checkpoint GC pod spec.
      This configuration is done using the ``task_container_defaults.checkpointGcPodSpec`` field
      within your ``value.yaml`` file. User can create a custom pod specification for CheckpointGC,
      it will override the default experiment's pod spec settings. Determined by default uses the
      experiment's pod spec, but by providing custom pod spec users have the flexibility to
      customize and configure the pod spec directly in this field. User can tailor the garbage
      collection settings according to the specific GC needs.
